package apifu

import (
	"context"
	"encoding/base64"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/vmihailenco/msgpack"

	"github.com/ccbrown/api-fu/graphql"
	"github.com/ccbrown/api-fu/graphql/schema"
	"github.com/ccbrown/api-fu/pagination"
)

type ConnectionDirection int

const (
	ConnectionDirectionBidirectional ConnectionDirection = iota
	ConnectionDirectionForwardOnly
	ConnectionDirectionBackwardOnly
)

// ConnectionConfig defines the configuration for a connection that adheres to the GraphQL Cursor
// Connections Specification.
type ConnectionConfig struct {
	// A prefix to use for the connection and edge type names. For example, if you provide
	// "Example", the connection type will be named "ExampleConnection" and the edge type will be
	// "ExampleEdge".
	NamePrefix string

	// An optional description for the connection.
	Description string

	// An optional deprecation reason for the connection.
	DeprecationReason string

	// The direction of the connection. This determines which of the first/last/before/after
	// arguments are defined on the connection.
	Direction ConnectionDirection

	// An optional map of additional arguments to add to the connection.
	Arguments map[string]*graphql.InputValueDefinition

	// If getting all edges for the connection is cheap, you can just provide ResolveAllEdges.
	// ResolveAllEdges should return a slice value, with one item for each edge, and a function that
	// can be used to sort the cursors produced by EdgeCursor.
	ResolveAllEdges func(ctx graphql.FieldContext) (edgeSlice any, cursorLess func(a, b any) bool, err error)

	// If getting all edges for the connection is too expensive for ResolveAllEdges, you can provide
	// ResolveEdges. ResolveEdges is just like ResolveAllEdges, but is only required to return edges
	// within the range defined by the given cursors and is only required to return up to `limit`
	// edges. If limit is negative, the last edges within the range should be returned instead of
	// the first.
	//
	// Returning extra edges or out-of-order edges is fine. They will be sorted and filtered
	// automatically. However, you should ensure that no duplicate edges are returned.
	//
	// If desired, edges outside of the given range may be returned to indicate the presence of more
	// pages before or after the given range. This is completely optional, and the connection's
	// behavior will be fully compliant with the Relay Pagination spec regardless. However,
	// providing these additional edges will allow hasNextPage and hasPreviousPage to be true in
	// scenarios where the spec allows them to be false for performance reasons.
	ResolveEdges func(ctx graphql.FieldContext, after, before any, limit int) (edgeSlice any, cursorLess func(a, b any) bool, err error)

	// If you use ResolveEdges, you can optionally provide ResolveTotalCount to add a totalCount
	// field to the connection. If you use ResolveAllEdges, there is no need to provide this.
	ResolveTotalCount func(ctx graphql.FieldContext) (any, error)

	// CursorType allows the connection to deserialize cursors. It is required for all connections.
	CursorType reflect.Type

	// EdgeCursor should return a value that can be used to determine the edge's relative ordering.
	// For example, this might be a struct with a name and id for a connection whose edges are
	// sorted by name. The value must be able to be marshaled to and from binary. This function
	// should return the type of cursor assigned to CursorType.
	EdgeCursor func(edge any) any

	// EdgeFields should provide definitions for the fields of each node. You must provide the
	// "node" field, but the "cursor" field will be provided for you.
	EdgeFields map[string]*graphql.FieldDefinition

	// The connection will implement these interfaces. If any of the interfaces define an edge
	// field as an interface, this connection's edges will also implement that interface.
	ImplementedInterfaces []*graphql.InterfaceType

	// This connection is only available for introspection and use when the given features are enabled.
	RequiredFeatures graphql.FeatureSet
}

// SerializeCursor serializes a cursor to a string that can be used in a response.
func SerializeCursor(cursor any) (string, error) {
	b, err := msgpack.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// DeserializeCursor deserializes a cursor that was previously serialized with SerializeCursor or
// returns nil if the cursor is invalid.
func DeserializeCursor(t reflect.Type, s string) any {
	ret := reflect.New(t)
	if b, err := base64.RawURLEncoding.DecodeString(s); err == nil {
		if err := msgpack.Unmarshal(b, ret.Interface()); err == nil {
			return ret.Elem().Interface()
		}
	}
	return nil
}

// PageInfo represents the page info of a GraphQL Cursor Connection.
type PageInfo struct {
	HasPreviousPage bool
	HasNextPage     bool
	StartCursor     string
	EndCursor       string
}

// PageInfoType implements the GraphQL type for the page info of a GraphQL Cursor Connection.
var PageInfoType = &graphql.ObjectType{
	Name: "PageInfo",
	Fields: map[string]*graphql.FieldDefinition{
		"hasPreviousPage": {
			Type:        graphql.NewNonNullType(graphql.BooleanType),
			Cost:        graphql.FieldResolverCost(0),
			Description: "When paginating backwards, this field indicates whether there are additional pages before the current one.",
			Resolve: func(ctx graphql.FieldContext) (any, error) {
				return ctx.Object.(*PageInfo).HasPreviousPage, nil
			},
		},
		"hasNextPage": {
			Type:        graphql.NewNonNullType(graphql.BooleanType),
			Cost:        graphql.FieldResolverCost(0),
			Description: "When paginating forwards, this field indicates whether there are additional pages after the current one.",
			Resolve: func(ctx graphql.FieldContext) (any, error) {
				return ctx.Object.(*PageInfo).HasNextPage, nil
			},
		},
		"startCursor": {
			// XXX: In the latest Relay spec, this is nullable
			// (https://github.com/facebook/relay/pull/2655). However, it would technically be a
			// breaking change to make it nullable here now.
			// TODO: Update them if there's an opportunity to safely do so.
			Type:        graphql.NewNonNullType(graphql.StringType),
			Cost:        graphql.FieldResolverCost(0),
			Description: "This is the cursor of the first edge in the current page.",
			Resolve: func(ctx graphql.FieldContext) (any, error) {
				return ctx.Object.(*PageInfo).StartCursor, nil
			},
		},
		"endCursor": {
			// XXX: See note on startCursor.
			Type:        graphql.NewNonNullType(graphql.StringType),
			Cost:        graphql.FieldResolverCost(0),
			Description: "This is the cursor of the last edge in the current page.",
			Resolve: func(ctx graphql.FieldContext) (any, error) {
				return ctx.Object.(*PageInfo).EndCursor, nil
			},
		},
	},
}

// Defines the configuration for a connection interface.
type ConnectionInterfaceConfig struct {
	// A prefix to use for the connection and edge type names. For example, if you provide
	// "Example", the connection type will be named "ExampleConnection" and the edge type will be
	// "ExampleEdge".
	NamePrefix string

	// EdgeFields should provide definitions for the fields of each node. You must provide the
	// "node" field, but the "cursor" field will be provided for you.
	EdgeFields map[string]*graphql.FieldDefinition

	// If true, implementations must provide the "totalCount" field.
	HasTotalCount bool

	// This connection is only available for introspection and use when the given features are enabled.
	RequiredFeatures graphql.FeatureSet
}

var forwardConnectionArguments = map[string]*graphql.InputValueDefinition{
	"first": {
		Type:        graphql.NewNonNullType(graphql.IntType),
		Description: "Indicates that up to the first N results should be returned.",
	},
	"after": {
		Type:        graphql.StringType,
		Description: "Returns only results that come after the given cursor.",
	},
}

var backwardConnectionArguments = map[string]*graphql.InputValueDefinition{
	"last": {
		Type:        graphql.NewNonNullType(graphql.IntType),
		Description: "Indicates that up to the last N results should be returned.",
	},
	"before": {
		Type:        graphql.StringType,
		Description: "Returns only results that come before the given cursor.",
	},
}

var bidirectionalConnectionArguments = map[string]*graphql.InputValueDefinition{
	"first": {
		Type:        graphql.IntType,
		Description: "Indicates that up to the first N results should be returned. You must provide either `first` or `last`.",
	},
	"after": {
		Type:        graphql.StringType,
		Description: "Returns only results that come after the given cursor.",
	},
	"last": {
		Type:        graphql.IntType,
		Description: "Indicates that up to the last N results should be returned. You must provide either `first` or `last`.",
	},
	"before": {
		Type:        graphql.StringType,
		Description: "Returns only results that come before the given cursor.",
	},
}

func defaultConnectionCost(ctx graphql.FieldCostContext) graphql.FieldCost {
	maxCount, _ := ctx.Arguments["first"].(int)
	if last, ok := ctx.Arguments["last"].(int); ok {
		maxCount = last
	}
	return graphql.FieldCost{
		Context:  context.WithValue(ctx.Context, maxEdgeCountContextKey, maxCount),
		Resolver: 1,
	}
}

const cursorDesc = "A cursor for pagination via a connection's `before` and `after` arguments. Cursors are opaque strings and are not meant to be used by clients except to paginate through a result set."
const pageInfoDesc = "Information about the current page of results."
const totalCountDesc = "The total count of existing items, including those not returned in the current page."
const edgesDesc = `A list of edges. An edge represents a relationship with a "node", and may include additional fields describing that relationship.`

// Returns an interface for a connection.
func ConnectionInterface(config *ConnectionInterfaceConfig) *graphql.InterfaceType {
	edgeFields := map[string]*graphql.FieldDefinition{
		"cursor": &graphql.FieldDefinition{
			Type:        graphql.NewNonNullType(graphql.StringType),
			Cost:        graphql.FieldResolverCost(0),
			Description: cursorDesc,
		},
	}
	for k, v := range config.EdgeFields {
		edgeFields[k] = v
	}

	edge := &graphql.InterfaceType{
		Name:             config.NamePrefix + "Edge",
		Fields:           edgeFields,
		RequiredFeatures: config.RequiredFeatures,
	}

	ret := &graphql.InterfaceType{
		Name:             config.NamePrefix + "Connection",
		RequiredFeatures: config.RequiredFeatures,
		Fields: map[string]*graphql.FieldDefinition{
			"edges": {
				Type:        graphql.NewNonNullType(graphql.NewListType(graphql.NewNonNullType(edge))),
				Description: edgesDesc,
				Cost: func(ctx graphql.FieldCostContext) graphql.FieldCost {
					return graphql.FieldCost{
						Resolver:   0,
						Multiplier: ctx.Context.Value(maxEdgeCountContextKey).(int),
					}
				},
			},
			"pageInfo": {
				Type: graphql.NewNonNullType(PageInfoType),
				// The cost is already accounted for by the connection itself. Either
				// ResolvePageInfo will be trivial or 0 edges were requested and all work was
				// delayed until now.
				Cost:        graphql.FieldResolverCost(0),
				Description: pageInfoDesc,
			},
		},
	}

	if config.HasTotalCount {
		ret.Fields["totalCount"] = &graphql.FieldDefinition{
			Type:        graphql.NewNonNullType(graphql.IntType),
			Description: totalCountDesc,
		}
	}

	return ret
}

// Defines the configuration for a connection interface.
type ConnectionFieldDefinitionConfig struct {
	// The type of the connection.
	Type graphql.Type

	// The direction of the connection.
	Direction ConnectionDirection

	// An optional description for the connection field.
	Description string

	// An optional deprecation reason for the connection field.
	DeprecationReason string

	// An optional map of additional arguments to add to the field.
	Arguments map[string]*graphql.InputValueDefinition

	// This connection is only available for introspection and use when the given features are enabled.
	RequiredFeatures graphql.FeatureSet
}

// Returns a minimal connection field definition, with default arguments and cost function defined.
func ConnectionFieldDefinition(config *ConnectionFieldDefinitionConfig) *graphql.FieldDefinition {
	ret := &graphql.FieldDefinition{
		Type:              config.Type,
		Arguments:         map[string]*graphql.InputValueDefinition{},
		Cost:              defaultConnectionCost,
		Description:       config.Description,
		DeprecationReason: config.DeprecationReason,
		RequiredFeatures:  config.RequiredFeatures,
	}
	switch config.Direction {
	case ConnectionDirectionForwardOnly:
		for name, def := range forwardConnectionArguments {
			ret.Arguments[name] = def
		}
	case ConnectionDirectionBackwardOnly:
		for name, def := range backwardConnectionArguments {
			ret.Arguments[name] = def
		}
	case ConnectionDirectionBidirectional:
		for name, def := range bidirectionalConnectionArguments {
			ret.Arguments[name] = def
		}
	}
	for name, def := range config.Arguments {
		ret.Arguments[name] = def
	}
	return ret
}

type edge struct {
	value    any
	cursor   userCursor
	typeName string
}

func (e edge) Cursor() userCursor {
	return e.cursor
}

type userCursor struct {
	value      any
	cursorLess func(a, b any) bool
}

func (c userCursor) LessThan(other userCursor) bool {
	return c.cursorLess(c.value, other.value)
}

type connection struct {
	ResolveTotalCount func() (any, error)
	Edges             []edge
	ResolvePageInfo   func() (any, error)
	typeName          string
}

type maxEdgeCountContextKeyType int

var maxEdgeCountContextKey maxEdgeCountContextKeyType

// Connection is used to create a connection field that adheres to the GraphQL Cursor Connections
// Specification.
func Connection(config *ConnectionConfig) *graphql.FieldDefinition {
	edgeFields := map[string]*graphql.FieldDefinition{
		"cursor": {
			Type:        graphql.NewNonNullType(graphql.StringType),
			Cost:        graphql.FieldResolverCost(0),
			Description: cursorDesc,
			Resolve: func(ctx graphql.FieldContext) (any, error) {
				s, err := SerializeCursor(ctx.Object.(edge).cursor.value)
				if err != nil {
					return nil, errors.Wrap(err, "error serializing cursor")
				}
				return s, nil
			},
		},
	}
	for k, v := range config.EdgeFields {
		def := *v
		resolve := def.Resolve
		def.Resolve = func(ctx graphql.FieldContext) (any, error) {
			ctx.Object = ctx.Object.(edge).value
			return resolve(ctx)
		}
		edgeFields[k] = &def
	}

	edgeType := &graphql.ObjectType{
		Name:             config.NamePrefix + "Edge",
		Fields:           edgeFields,
		RequiredFeatures: config.RequiredFeatures,
		IsTypeOf: func(obj any) bool {
			e, ok := obj.(edge)
			return ok && e.typeName == config.NamePrefix+"Edge"
		},
	}
	for _, iface := range config.ImplementedInterfaces {
		if ifaceEdge, ok := iface.Fields["edges"]; ok {
			if edgeInterface, ok := schema.UnwrappedType(ifaceEdge.Type).(*graphql.InterfaceType); ok {
				edgeType.ImplementedInterfaces = append(edgeType.ImplementedInterfaces, edgeInterface)
			}
		}
	}

	connectionType := &graphql.ObjectType{
		Name:             config.NamePrefix + "Connection",
		Description:      config.Description,
		RequiredFeatures: config.RequiredFeatures,
		Fields: map[string]*graphql.FieldDefinition{
			"edges": {
				Type: graphql.NewNonNullType(graphql.NewListType(graphql.NewNonNullType(edgeType))),
				Cost: func(ctx graphql.FieldCostContext) graphql.FieldCost {
					return graphql.FieldCost{
						Resolver:   0,
						Multiplier: ctx.Context.Value(maxEdgeCountContextKey).(int),
					}
				},
				Description: edgesDesc,
				Resolve: func(ctx graphql.FieldContext) (any, error) {
					return ctx.Object.(*connection).Edges, nil
				},
			},
			"pageInfo": {
				Type: graphql.NewNonNullType(PageInfoType),
				// The cost is already accounted for by the connection itself. Either
				// ResolvePageInfo will be trivial or 0 edges were requested and all work was
				// delayed until now.
				Cost:        graphql.FieldResolverCost(0),
				Description: pageInfoDesc,
				Resolve: func(ctx graphql.FieldContext) (any, error) {
					return ctx.Object.(*connection).ResolvePageInfo()
				},
			},
		},
		ImplementedInterfaces: config.ImplementedInterfaces,
		IsTypeOf: func(obj any) bool {
			c, ok := obj.(*connection)
			return ok && c.typeName == config.NamePrefix+"Connection"
		},
	}

	if config.ResolveAllEdges != nil || config.ResolveTotalCount != nil {
		connectionType.Fields["totalCount"] = &graphql.FieldDefinition{
			Type:        graphql.NewNonNullType(graphql.IntType),
			Description: totalCountDesc,
			Resolve: func(ctx graphql.FieldContext) (any, error) {
				return ctx.Object.(*connection).ResolveTotalCount()
			},
		}
	}

	ret := ConnectionFieldDefinition(&ConnectionFieldDefinitionConfig{
		Type:              connectionType,
		Direction:         config.Direction,
		Description:       config.Description,
		DeprecationReason: config.DeprecationReason,
		Arguments:         config.Arguments,
		RequiredFeatures:  config.RequiredFeatures,
	})
	ret.Resolve = func(ctx graphql.FieldContext) (any, error) {
		if first, ok := ctx.Arguments["first"].(int); ok {
			if first < 0 {
				return nil, fmt.Errorf("The `first` argument cannot be negative.")
			} else if _, ok := ctx.Arguments["last"].(int); ok {
				return nil, fmt.Errorf("You cannot provide both `first` and `last` arguments.")
			}
		} else if last, ok := ctx.Arguments["last"].(int); ok {
			if last < 0 {
				return nil, fmt.Errorf("The `last` argument cannot be negative.")
			}
		} else {
			return nil, fmt.Errorf("You must provide either the `first` or `last` argument.")
		}

		var afterCursor, beforeCursor any

		if after, _ := ctx.Arguments["after"].(string); after != "" {
			if value := DeserializeCursor(config.CursorType, after); value == nil {
				return nil, fmt.Errorf("Invalid after cursor.")
			} else {
				afterCursor = value
			}
		}

		if before, _ := ctx.Arguments["before"].(string); before != "" {
			if value := DeserializeCursor(config.CursorType, before); value == nil {
				return nil, fmt.Errorf("Invalid before cursor.")
			} else {
				beforeCursor = value
			}
		}

		var limit int
		if first, ok := ctx.Arguments["first"].(int); ok {
			limit = first + 1
		} else {
			limit = -(ctx.Arguments["last"].(int) + 1)
		}
		resolve := func() (any, func(a, b any) bool, error) {
			return config.ResolveAllEdges(ctx)
		}
		if config.ResolveAllEdges == nil {
			resolve = func() (any, func(a, b any) bool, error) {
				return config.ResolveEdges(ctx, afterCursor, beforeCursor, limit)
			}
		}
		if limit == 1 || limit == -1 {
			// no edges. don't do anything unless pageInfo is requested
			return &connection{
				ResolveTotalCount: func() (any, error) {
					return config.ResolveTotalCount(ctx)
				},
				Edges: []edge{},
				ResolvePageInfo: func() (any, error) {
					edgeSlice, cursorLess, err := resolve()
					if !isNil(err) {
						return nil, err
					}
					conn, err := completeConnection(config, ctx, beforeCursor, afterCursor, cursorLess, edgeSlice)
					if !isNil(err) {
						return nil, err
					}
					if promise, ok := conn.(graphql.ResolvePromise); ok {
						return chain(ctx.Context, promise, func(conn any) (any, error) {
							return conn.(*connection).ResolvePageInfo()
						}), nil
					}
					return conn.(*connection).ResolvePageInfo()
				},
			}, nil
		}
		edgeSlice, cursorLess, err := resolve()
		if !isNil(err) {
			return nil, err
		}
		return completeConnection(config, ctx, beforeCursor, afterCursor, cursorLess, edgeSlice)
	}
	return ret
}

func completeConnection(config *ConnectionConfig, ctx graphql.FieldContext, beforeCursorValue, afterCursorValue any, cursorLess func(a, b any) bool, edgeSlice any) (any, error) {
	if edgeSlice, ok := edgeSlice.(graphql.ResolvePromise); ok {
		return chain(ctx.Context, edgeSlice, func(edgeSlice any) (any, error) {
			return completeConnection(config, ctx, beforeCursorValue, afterCursorValue, cursorLess, edgeSlice)
		}), nil
	}

	edgeSliceValue := reflect.ValueOf(edgeSlice)
	if edgeSliceValue.Kind() != reflect.Slice {
		return nil, fmt.Errorf("unexpected non-slice type %T for edges", edgeSlice)
	}

	resolveTotalCount := func() (any, error) {
		return edgeSliceValue.Len(), nil
	}
	if config.ResolveTotalCount != nil {
		resolveTotalCount = func() (any, error) {
			return config.ResolveTotalCount(ctx)
		}
	}

	edgesWithCursors := make([]edge, edgeSliceValue.Len())
	for i := range edgesWithCursors {
		value := edgeSliceValue.Index(i).Interface()
		edgesWithCursors[i] = edge{
			value: value,
			cursor: userCursor{
				value:      config.EdgeCursor(value),
				cursorLess: cursorLess,
			},
			typeName: config.NamePrefix + "Edge",
		}
	}

	var afterCursor, beforeCursor *userCursor
	if afterCursorValue != nil {
		afterCursor = &userCursor{
			value:      afterCursorValue,
			cursorLess: cursorLess,
		}
	}
	if beforeCursorValue != nil {
		beforeCursor = &userCursor{
			value:      beforeCursorValue,
			cursorLess: cursorLess,
		}
	}

	var first, last *int
	if f, ok := ctx.Arguments["first"].(int); ok {
		first = &f
	}
	if l, ok := ctx.Arguments["last"].(int); ok {
		last = &l
	}

	edges, pageInfo := pagination.EdgesToReturn(edgesWithCursors, afterCursor, beforeCursor, first, last)

	serializedPageInfo := &PageInfo{
		HasPreviousPage: pageInfo.HasPreviousPage,
		HasNextPage:     pageInfo.HasNextPage,
	}
	if len(edges) > 0 {
		var err error
		serializedPageInfo.StartCursor, err = SerializeCursor(pageInfo.StartCursor.value)
		if err != nil {
			return nil, errors.Wrap(err, "error serializing start cursor")
		}
		serializedPageInfo.EndCursor, err = SerializeCursor(pageInfo.EndCursor.value)
		if err != nil {
			return nil, errors.Wrap(err, "error serializing end cursor")
		}
	}
	return &connection{
		ResolveTotalCount: resolveTotalCount,
		Edges:             edges,
		ResolvePageInfo: func() (any, error) {
			return serializedPageInfo, nil
		},
	}, nil
}

// TimeBasedCursor represents the data embedded in cursors for time-based connections.
type TimeBasedCursor struct {
	Nano int64
	Id   string
}

// NewTimeBasedCursor constructs a TimeBasedCursor.
func NewTimeBasedCursor(t time.Time, id string) TimeBasedCursor {
	return TimeBasedCursor{
		Nano: t.UnixNano(),
		Id:   id,
	}
}

func (c TimeBasedCursor) Time() time.Time {
	return time.Unix(0, c.Nano)
}

func (c TimeBasedCursor) LessThan(other TimeBasedCursor) bool {
	return c.Nano < other.Nano || (c.Nano == other.Nano && strings.Compare(c.Id, other.Id) < 0)
}

func timeBasedCursorLess(a, b any) bool {
	return a.(TimeBasedCursor).LessThan(b.(TimeBasedCursor))
}

// TimeBasedConnectionConfig defines the configuration for a time-based connection that adheres to
// the GraphQL Cursor Connections Specification.
type TimeBasedConnectionConfig struct {
	// An optional description for the connection.
	Description string

	// An optional deprecation reason for the connection.
	DeprecationReason string

	// A required prefix for the type names. For a field named "friendsConnection" on a User type,
	// the recommended prefix would be "UserFriends". This will result in types named
	// "UserFriendsConnection" and "UserFriendsEdge".
	NamePrefix string

	// This function should return a TimeBasedCursor for the given edge.
	EdgeCursor func(edge any) TimeBasedCursor

	// Returns the fields for the edge. This should always at least include a "node" field.
	EdgeFields map[string]*graphql.FieldDefinition

	// The getter for the edges. If limit is zero, all edges within the given range should be
	// returned. If limit is greater than zero, up to limit edges at the start of the range should
	// be returned. If limit is less than zero, up to -limit edge at the end of the range should be
	// returned.
	EdgeGetter func(ctx graphql.FieldContext, minTime time.Time, maxTime time.Time, limit int) (any, error)

	// An optional map of additional arguments to add to the connection.
	Arguments map[string]*graphql.InputValueDefinition

	// To support the "totalCount" connection field, you can provide this method.
	ResolveTotalCount func(ctx graphql.FieldContext) (any, error)

	// The connection will implement these interfaces. If any of the interfaces define an edge
	// field as an interface, this connection's edges will also implement that interface.
	ImplementedInterfaces []*graphql.InterfaceType

	// This connection is only available for introspection and use when the given features are enabled.
	RequiredFeatures graphql.FeatureSet
}

// TimeBasedConnection creates a new connection for edges sorted by time. In addition to the
// standard first, last, after, and before fields, the connection will have atOrAfterTime and
// beforeTime fields, which can be used to query a specific time range.
func TimeBasedConnection(config *TimeBasedConnectionConfig) *graphql.FieldDefinition {
	arguments := map[string]*graphql.InputValueDefinition{
		"atOrAfterTime": {
			Type:        DateTimeType,
			Description: "Filters results such that only those that occurred at or after this time are returned.",
		},
		"beforeTime": {
			Type:        DateTimeType,
			Description: "Filters results such that only those that occurred before this time are returned.",
		},
	}
	for name, def := range config.Arguments {
		arguments[name] = def
	}

	description := "Provides nodes sorted by time."
	if config.Description != "" {
		description = config.Description
	}

	return Connection(&ConnectionConfig{
		NamePrefix:        config.NamePrefix,
		Arguments:         arguments,
		Description:       description,
		DeprecationReason: config.DeprecationReason,
		EdgeCursor: func(edge any) any {
			return config.EdgeCursor(edge)
		},
		EdgeFields:        config.EdgeFields,
		RequiredFeatures:  config.RequiredFeatures,
		CursorType:        reflect.TypeOf(TimeBasedCursor{}),
		ResolveTotalCount: config.ResolveTotalCount,
		ResolveEdges: func(ctx graphql.FieldContext, after, before any, limit int) (edgeSlice any, cursorLess func(a, b any) bool, err error) {
			var atOrAfterTime, beforeTime *time.Time
			if t, ok := ctx.Arguments["atOrAfterTime"].(time.Time); ok {
				atOrAfterTime = &t
			}
			if t, ok := ctx.Arguments["beforeTime"].(time.Time); ok {
				beforeTime = &t
			}

			var afterPtr, beforePtr *TimeBasedCursor
			if c, ok := after.(TimeBasedCursor); ok {
				afterPtr = &c
			}
			if c, ok := before.(TimeBasedCursor); ok {
				beforePtr = &c
			}

			queries := pagination.TimeBasedRangeQueries(afterPtr, beforePtr, atOrAfterTime, beforeTime, limit)

			var edges []any
			var promises []graphql.ResolvePromise
			for _, q := range queries {
				if queryEdges, err := config.EdgeGetter(ctx, q.MinTime, q.MaxTime, q.Limit); err != nil {
					return nil, nil, err
				} else if promise, ok := queryEdges.(graphql.ResolvePromise); ok {
					promises = append(promises, promise)
				} else {
					v := reflect.ValueOf(queryEdges)
					if v.Kind() == reflect.Invalid || v.IsNil() {
						continue
					}
					for i := 0; i < v.Len(); i++ {
						edges = append(edges, v.Index(i).Interface())
					}
				}
			}
			if len(promises) > 0 {
				return join(ctx.Context, promises, func(v []any) (any, error) {
					for _, queryEdges := range v {
						v := reflect.ValueOf(queryEdges)
						for i := 0; i < v.Len(); i++ {
							edges = append(edges, v.Index(i).Interface())
						}
					}
					return edges, nil
				}), timeBasedCursorLess, err
			}
			return edges, timeBasedCursorLess, err
		},
		ImplementedInterfaces: config.ImplementedInterfaces,
	})
}
