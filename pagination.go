package apifu

import (
	"context"
	"encoding/base64"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/vmihailenco/msgpack"

	"github.com/ccbrown/api-fu/graphql"
	"github.com/ccbrown/api-fu/graphql/schema"
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

	// An optional map of additional arguments to add to the connection.
	Arguments map[string]*graphql.InputValueDefinition

	// If getting all edges for the connection is cheap, you can just provide ResolveAllEdges.
	// ResolveAllEdges should return a slice value, with one item for each edge, and a function that
	// can be used to sort the cursors produced by EdgeCursor.
	ResolveAllEdges func(ctx *graphql.FieldContext) (edgeSlice interface{}, cursorLess func(a, b interface{}) bool, err error)

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
	ResolveEdges func(ctx *graphql.FieldContext, after, before interface{}, limit int) (edgeSlice interface{}, cursorLess func(a, b interface{}) bool, err error)

	// If you use ResolveEdges, you can optionally provide ResolveTotalCount to add a totalCount
	// field to the connection. If you use ResolveAllEdges, there is no need to provide this.
	ResolveTotalCount func(ctx *graphql.FieldContext) (interface{}, error)

	// CursorType allows the connection to deserialize cursors. It is required for all connections.
	CursorType reflect.Type

	// EdgeCursor should return a value that can be used to determine the edge's relative ordering.
	// For example, this might be a struct with a name and id for a connection whose edges are
	// sorted by name. The value must be able to be marshaled to and from binary. This function
	// should return the type of cursor assigned to CursorType.
	EdgeCursor func(edge interface{}) interface{}

	// EdgeFields should provide definitions for the fields of each node. You must provide the
	// "node" field, but the "cursor" field will be provided for you.
	EdgeFields map[string]*graphql.FieldDefinition

	// The connection will implement these interfaces. If any of the interfaces define an edge
	// field as an interface, this connection's edges will also implement that interface.
	ImplementedInterfaces []*graphql.InterfaceType
}

func serializeCursor(cursor interface{}) (string, error) {
	b, err := msgpack.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func deserializeCursor(t reflect.Type, s string) interface{} {
	ret := reflect.New(t)
	if b, err := base64.RawURLEncoding.DecodeString(s); err == nil {
		if err := msgpack.Unmarshal(b, ret.Interface()); err == nil {
			return ret.Elem().Interface()
		}
	}
	return nil
}

func (cfg *ConnectionConfig) applyCursorsToEdges(allEdges []interface{}, before, after interface{}, cursorLess func(a, b interface{}) bool) (edges []edge, hasPreviousPage, hasNextPage bool) {
	edges = []edge{}

	if len(allEdges) == 0 {
		return edges, false, false
	}

	for _, e := range allEdges {
		cursor := cfg.EdgeCursor(e)
		if after != nil && !cursorLess(after, cursor) {
			hasPreviousPage = true
			continue
		}
		if before != nil && !cursorLess(cursor, before) {
			hasNextPage = true
			continue
		}
		edges = append(edges, edge{
			Value:  e,
			Cursor: cursor,
		})
	}

	sort.Slice(edges, func(i, j int) bool {
		return cursorLess(edges[i].Cursor, edges[j].Cursor)
	})

	return
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
		"hasPreviousPage": NonNull(graphql.BooleanType, "HasPreviousPage"),
		"hasNextPage":     NonNull(graphql.BooleanType, "HasNextPage"),
		"startCursor":     NonNull(graphql.StringType, "StartCursor"),
		"endCursor":       NonNull(graphql.StringType, "EndCursor"),
	},
}

type edge struct {
	Value  interface{}
	Cursor interface{}
}

type connection struct {
	ResolveTotalCount func() (interface{}, error)
	Edges             []edge
	ResolvePageInfo   func() (interface{}, error)
}

type maxEdgeCountContextKeyType int

var maxEdgeCountContextKey maxEdgeCountContextKeyType

// Connection is used to create a connection field that adheres to the GraphQL Cursor Connections
// Specification.
func Connection(config *ConnectionConfig) *graphql.FieldDefinition {
	edgeFields := map[string]*graphql.FieldDefinition{
		"cursor": {
			Type: graphql.NewNonNullType(graphql.StringType),
			Cost: graphql.FieldResolverCost(0),
			Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
				s, err := serializeCursor(ctx.Object.(edge).Cursor)
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
		def.Resolve = func(ctxIn *graphql.FieldContext) (interface{}, error) {
			ctx := *ctxIn
			ctx.Object = ctxIn.Object.(edge).Value
			return resolve(&ctx)
		}
		edgeFields[k] = &def
	}

	edgeType := &graphql.ObjectType{
		Name:   config.NamePrefix + "Edge",
		Fields: edgeFields,
	}
	for _, iface := range config.ImplementedInterfaces {
		if ifaceEdge, ok := iface.Fields["edges"]; ok {
			if edgeInterface, ok := schema.UnwrappedType(ifaceEdge.Type).(*graphql.InterfaceType); ok {
				edgeType.ImplementedInterfaces = append(edgeType.ImplementedInterfaces, edgeInterface)
			}
		}
	}

	connectionType := &graphql.ObjectType{
		Name:        config.NamePrefix + "Connection",
		Description: config.Description,
		Fields: map[string]*graphql.FieldDefinition{
			"edges": {
				Type: graphql.NewNonNullType(graphql.NewListType(graphql.NewNonNullType(edgeType))),
				Cost: func(ctx *graphql.FieldCostContext) graphql.FieldCost {
					return graphql.FieldCost{
						Resolver:   0,
						Multiplier: ctx.Context.Value(maxEdgeCountContextKey).(int),
					}
				},
				Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
					return ctx.Object.(*connection).Edges, nil
				},
			},
			"pageInfo": {
				Type: graphql.NewNonNullType(PageInfoType),
				// The cost is already accounted for by the connection itself. Either
				// ResolvePageInfo will be trivial or 0 edges were requested and all work was
				// delayed until now.
				Cost: graphql.FieldResolverCost(0),
				Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
					return ctx.Object.(*connection).ResolvePageInfo()
				},
			},
		},
		ImplementedInterfaces: config.ImplementedInterfaces,
	}

	if config.ResolveAllEdges != nil || config.ResolveTotalCount != nil {
		connectionType.Fields["totalCount"] = &graphql.FieldDefinition{
			Type: graphql.NewNonNullType(graphql.IntType),
			Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
				return ctx.Object.(*connection).ResolveTotalCount()
			},
		}
	}

	ret := &graphql.FieldDefinition{
		Type: connectionType,
		Arguments: map[string]*graphql.InputValueDefinition{
			"first": {
				Type: graphql.IntType,
			},
			"last": {
				Type: graphql.IntType,
			},
			"before": {
				Type: graphql.StringType,
			},
			"after": {
				Type: graphql.StringType,
			},
		},
		Cost: func(ctx *graphql.FieldCostContext) graphql.FieldCost {
			maxCount, _ := ctx.Arguments["first"].(int)
			if last, ok := ctx.Arguments["last"].(int); ok {
				maxCount = last
			}
			return graphql.FieldCost{
				Context:  context.WithValue(ctx.Context, maxEdgeCountContextKey, maxCount),
				Resolver: 1,
			}
		},
		Description: config.Description,
		Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
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

			var afterCursor interface{}
			if after, _ := ctx.Arguments["after"].(string); after != "" {
				if afterCursor = deserializeCursor(config.CursorType, after); afterCursor == nil {
					return nil, fmt.Errorf("Invalid after cursor.")
				}
			}

			var beforeCursor interface{}
			if before, _ := ctx.Arguments["before"].(string); before != "" {
				if beforeCursor = deserializeCursor(config.CursorType, before); beforeCursor == nil {
					return nil, fmt.Errorf("Invalid before cursor.")
				}
			}

			var limit int
			if first, ok := ctx.Arguments["first"].(int); ok {
				limit = first + 1
			} else {
				limit = -(ctx.Arguments["last"].(int) + 1)
			}
			resolve := func() (interface{}, func(a, b interface{}) bool, error) {
				return config.ResolveAllEdges(ctx)
			}
			if config.ResolveAllEdges == nil {
				resolve = func() (interface{}, func(a, b interface{}) bool, error) {
					return config.ResolveEdges(ctx, afterCursor, beforeCursor, limit)
				}
			}
			if limit == 1 || limit == -1 {
				// no edges. don't do anything unless pageInfo is requested
				return &connection{
					ResolveTotalCount: func() (interface{}, error) {
						return config.ResolveTotalCount(ctx)
					},
					Edges: []edge{},
					ResolvePageInfo: func() (interface{}, error) {
						edgeSlice, cursorLess, err := resolve()
						if !isNil(err) {
							return nil, err
						}
						conn, err := completeConnection(config, ctx, beforeCursor, afterCursor, cursorLess, edgeSlice)
						if !isNil(err) {
							return nil, err
						}
						if promise, ok := conn.(graphql.ResolvePromise); ok {
							return chain(ctx.Context, promise, func(conn interface{}) (interface{}, error) {
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
		},
	}

	for name, def := range config.Arguments {
		ret.Arguments[name] = def
	}

	return ret
}

func completeConnection(config *ConnectionConfig, ctx *graphql.FieldContext, beforeCursor, afterCursor interface{}, cursorLess func(a, b interface{}) bool, edgeSlice interface{}) (interface{}, error) {
	if edgeSlice, ok := edgeSlice.(graphql.ResolvePromise); ok {
		return chain(ctx.Context, edgeSlice, func(edgeSlice interface{}) (interface{}, error) {
			return completeConnection(config, ctx, beforeCursor, afterCursor, cursorLess, edgeSlice)
		}), nil
	}

	edgeSliceValue := reflect.ValueOf(edgeSlice)
	if edgeSliceValue.Kind() != reflect.Slice {
		return nil, fmt.Errorf("unexpected non-slice type %T for edges", edgeSlice)
	}

	resolveTotalCount := func() (interface{}, error) {
		return edgeSliceValue.Len(), nil
	}
	if config.ResolveTotalCount != nil {
		resolveTotalCount = func() (interface{}, error) {
			return config.ResolveTotalCount(ctx)
		}
	}

	ifaces := make([]interface{}, edgeSliceValue.Len())
	for i := range ifaces {
		ifaces[i] = edgeSliceValue.Index(i).Interface()
	}

	edges, hasPreviousPage, hasNextPage := config.applyCursorsToEdges(ifaces, beforeCursor, afterCursor, cursorLess)

	if first, ok := ctx.Arguments["first"].(int); ok {
		if len(edges) > first {
			edges = edges[:first]
			hasNextPage = true
		} else {
			hasNextPage = false
		}
	}

	if last, ok := ctx.Arguments["last"].(int); ok {
		if len(edges) > last {
			edges = edges[len(edges)-last:]
			hasPreviousPage = true
		} else {
			hasPreviousPage = false
		}
	}

	pageInfo := &PageInfo{
		HasPreviousPage: hasPreviousPage,
		HasNextPage:     hasNextPage,
	}
	if len(edges) > 0 {
		var err error
		pageInfo.StartCursor, err = serializeCursor(edges[0].Cursor)
		if err != nil {
			return nil, errors.Wrap(err, "error serializing start cursor")
		}
		pageInfo.EndCursor, err = serializeCursor(edges[len(edges)-1].Cursor)
		if err != nil {
			return nil, errors.Wrap(err, "error serializing end cursor")
		}
	}
	return &connection{
		ResolveTotalCount: resolveTotalCount,
		Edges:             edges,
		ResolvePageInfo: func() (interface{}, error) {
			return pageInfo, nil
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

func timeBasedCursorLess(a, b interface{}) bool {
	ac, bc := a.(TimeBasedCursor), b.(TimeBasedCursor)
	return ac.Nano < bc.Nano || (ac.Nano == bc.Nano && strings.Compare(ac.Id, bc.Id) < 0)
}

// TimeBasedConnectionConfig defines the configuration for a time-based connection that adheres to
// the GraphQL Cursor Connections Specification.
type TimeBasedConnectionConfig struct {
	// An optional description for the connection.
	Description string

	// A required prefix for the type names. For a field named "friendsConnection" on a User type,
	// the recommended prefix would be "UserFriends". This will result in types named
	// "UserFriendsConnection" and "UserFriendsEdge".
	NamePrefix string

	// This function should return a TimeBasedCursor for the given edge.
	EdgeCursor func(edge interface{}) TimeBasedCursor

	// Returns the fields for the edge. This should always at least include a "node" field.
	EdgeFields map[string]*graphql.FieldDefinition

	// The getter for the edges. If limit is zero, all edges within the given range should be
	// returned. If limit is greater than zero, up to limit edges at the start of the range should
	// be returned. If limit is less than zero, up to -limit edge at the end of the range should be
	// returned.
	EdgeGetter func(ctx *graphql.FieldContext, minTime time.Time, maxTime time.Time, limit int) (interface{}, error)

	// An optional map of additional arguments to add to the connection.
	Arguments map[string]*graphql.InputValueDefinition

	// To support the "totalCount" connection field, you can provide this method.
	ResolveTotalCount func(ctx *graphql.FieldContext) (interface{}, error)

	// The connection will implement these interfaces. If any of the interfaces define an edge
	// field as an interface, this connection's edges will also implement that interface.
	ImplementedInterfaces []*graphql.InterfaceType
}

var distantFuture = time.Date(3000, time.January, 1, 0, 0, 0, 0, time.UTC)

// TimeBasedConnection creates a new connection for edges sorted by time. In addition to the
// standard first, last, after, and before fields, the connection will have atOrAfterTime and
// beforeTime fields, which can be used to query a specific time range.
func TimeBasedConnection(config *TimeBasedConnectionConfig) *graphql.FieldDefinition {
	arguments := map[string]*graphql.InputValueDefinition{
		"atOrAfterTime": {
			Type: DateTimeType,
		},
		"beforeTime": {
			Type: DateTimeType,
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
		NamePrefix:  config.NamePrefix,
		Arguments:   arguments,
		Description: description,
		EdgeCursor: func(edge interface{}) interface{} {
			return config.EdgeCursor(edge)
		},
		EdgeFields:        config.EdgeFields,
		CursorType:        reflect.TypeOf(TimeBasedCursor{}),
		ResolveTotalCount: config.ResolveTotalCount,
		ResolveEdges: func(ctx *graphql.FieldContext, after, before interface{}, limit int) (edgeSlice interface{}, cursorLess func(a, b interface{}) bool, err error) {
			type Query struct {
				Min   time.Time
				Max   time.Time
				Limit int
			}
			var queries []Query

			atOrAfterTime := time.Time{}
			if t, ok := ctx.Arguments["atOrAfterTime"].(time.Time); ok {
				atOrAfterTime = t
			}

			beforeTime := distantFuture
			if t, ok := ctx.Arguments["beforeTime"].(time.Time); ok {
				beforeTime = t
			}

			middle := Query{atOrAfterTime, beforeTime.Add(-time.Nanosecond), limit}

			if after, ok := after.(TimeBasedCursor); ok {
				queries = append(queries, Query{time.Unix(0, after.Nano), time.Unix(0, after.Nano), 0})
				if t := time.Unix(0, after.Nano+1); t.After(middle.Min) {
					middle.Min = t
				}
			}

			if before, ok := before.(TimeBasedCursor); ok {
				if after, ok := after.(TimeBasedCursor); !ok || after.Nano != before.Nano {
					queries = append(queries, Query{time.Unix(0, before.Nano), time.Unix(0, before.Nano), 0})
				}
				if t := time.Unix(0, before.Nano-1); t.Before(middle.Max) {
					middle.Max = t
				}
			}

			queries = append(queries, middle)

			var edges []interface{}
			var promises []graphql.ResolvePromise
			for _, q := range queries {
				if queryEdges, err := config.EdgeGetter(ctx, q.Min, q.Max, q.Limit); err != nil {
					return nil, nil, err
				} else if promise, ok := queryEdges.(graphql.ResolvePromise); ok {
					promises = append(promises, promise)
				} else {
					v := reflect.ValueOf(queryEdges)
					for i := 0; i < v.Len(); i++ {
						edges = append(edges, v.Index(i).Interface())
					}
				}
			}
			if len(promises) > 0 {
				return join(ctx.Context, promises, func(v []interface{}) (interface{}, error) {
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
