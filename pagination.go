package apifu

import (
	"encoding/base64"
	"fmt"
	"reflect"
	"sort"

	"github.com/pkg/errors"
	"github.com/vmihailenco/msgpack"

	"github.com/ccbrown/api-fu/graphql"
)

type ConnectionConfig struct {
	// A prefix to use for the connection and edge type names. For example, if you provide
	// "Example", the connection type will be named "ExampleConnection" and the edge type will be
	// "ExampleEdge".
	NamePrefix string

	Description string

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

type PageInfo struct {
	HasPreviousPage bool
	HasNextPage     bool
	StartCursor     string
	EndCursor       string
}

var PageInfoType = &graphql.ObjectType{
	Name: "PageInfo",
	Fields: map[string]*graphql.FieldDefinition{
		"hasPreviousPage": NonNullBoolean("HasPreviousPage"),
		"hasNextPage":     NonNullBoolean("HasNextPage"),
		"startCursor":     NonNullString("StartCursor"),
		"endCursor":       NonNullString("EndCursor"),
	},
}

type edge struct {
	Value  interface{}
	Cursor interface{}
}

type connection struct {
	Edges    []edge
	PageInfo PageInfo
}

func Connection(config *ConnectionConfig) *graphql.FieldDefinition {
	edgeFields := map[string]*graphql.FieldDefinition{
		"cursor": &graphql.FieldDefinition{
			Type: graphql.NewNonNullType(graphql.StringType),
			Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
				if s, err := serializeCursor(ctx.Object.(edge).Cursor); err != nil {
					return nil, errors.Wrap(err, "error serializing cursor")
				} else {
					return s, nil
				}
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

	connectionType := &graphql.ObjectType{
		Name:        config.NamePrefix + "Connection",
		Description: config.Description,
		Fields: map[string]*graphql.FieldDefinition{
			"edges": &graphql.FieldDefinition{
				Type: graphql.NewNonNullType(graphql.NewListType(graphql.NewNonNullType(edgeType))),
				Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
					return ctx.Object.(*connection).Edges, nil
				},
			},
			"pageInfo": &graphql.FieldDefinition{
				Type: graphql.NewNonNullType(PageInfoType),
				Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
					return ctx.Object.(*connection).PageInfo, nil
				},
			},
		},
	}

	return &graphql.FieldDefinition{
		Type: connectionType,
		Arguments: map[string]*graphql.InputValueDefinition{
			"first": &graphql.InputValueDefinition{
				Type: graphql.IntType,
			},
			"last": &graphql.InputValueDefinition{
				Type: graphql.IntType,
			},
			"before": &graphql.InputValueDefinition{
				Type: graphql.StringType,
			},
			"after": &graphql.InputValueDefinition{
				Type: graphql.StringType,
			},
		},
		Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
			if first, ok := ctx.Arguments["first"].(int); ok {
				if first < 0 {
					return nil, fmt.Errorf("The `first` argument cannot be negative.")
				} else if _, ok := ctx.Arguments["last"]; ok {
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

			var edgeSlice interface{}
			var cursorLess func(a, b interface{}) bool
			var err error
			if config.ResolveAllEdges != nil {
				edgeSlice, cursorLess, err = config.ResolveAllEdges(ctx)
			} else {
				var limit int
				if first, ok := ctx.Arguments["first"].(int); ok {
					limit = first + 1
				} else {
					limit = -(ctx.Arguments["last"].(int) + 1)
				}
				edgeSlice, cursorLess, err = config.ResolveEdges(ctx, afterCursor, beforeCursor, limit)
			}
			if !isNil(err) {
				return nil, err
			}

			edgeSliceValue := reflect.ValueOf(edgeSlice)
			if edgeSliceValue.Kind() != reflect.Slice {
				return nil, fmt.Errorf("unexpected non-slice type %T for edges", edgeSlice)
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

			ret := &connection{
				Edges: edges,
				PageInfo: PageInfo{
					HasPreviousPage: hasPreviousPage,
					HasNextPage:     hasNextPage,
				},
			}
			if len(edges) > 0 {
				var err error
				ret.PageInfo.StartCursor, err = serializeCursor(edges[0].Cursor)
				if err != nil {
					return nil, errors.Wrap(err, "error serializing start cursor")
				}
				ret.PageInfo.EndCursor, err = serializeCursor(edges[len(edges)-1].Cursor)
				if err != nil {
					return nil, errors.Wrap(err, "error serializing end cursor")
				}
			}
			return ret, nil
		},
	}
}
