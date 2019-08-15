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

	// EdgeCursor should return a value that can be used to determine the edge's relative ordering.
	// For example, this might be a struct with a name and id for a connection whose edges are
	// sorted by name. The value must be able to be marshaled to and from binary.
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

func (cfg *ConnectionConfig) applyCursorsToEdges(allEdges []interface{}, before, after string, cursorLess func(a, b interface{}) bool) (edges []edge, hasPreviousPage, hasNextPage bool) {
	edges = []edge{}

	if len(allEdges) == 0 {
		return edges, false, false
	}

	cursorType := reflect.TypeOf(cfg.EdgeCursor(allEdges[0]))

	var afterCursor interface{}
	if after != "" {
		afterCursor = deserializeCursor(cursorType, after)
	}

	var beforeCursor interface{}
	if before != "" {
		beforeCursor = deserializeCursor(cursorType, before)
	}

	for _, e := range allEdges {
		cursor := cfg.EdgeCursor(e)
		if afterCursor != nil && !cursorLess(afterCursor, cursor) {
			hasPreviousPage = true
			continue
		}
		if beforeCursor != nil && !cursorLess(cursor, beforeCursor) {
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
			if _, ok := ctx.Arguments["first"]; ok {
				if _, ok := ctx.Arguments["last"]; ok {
					return nil, fmt.Errorf("You cannot provide both `first` and `last` arguments.")
				}
			} else if _, ok := ctx.Arguments["last"]; !ok {
				return nil, fmt.Errorf("You must provide either the `first` or `last` argument.")
			}

			edgeSlice, cursorLess, err := config.ResolveAllEdges(ctx)
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

			before, _ := ctx.Arguments["before"].(string)
			after, _ := ctx.Arguments["after"].(string)
			edges, hasPreviousPage, hasNextPage := config.applyCursorsToEdges(ifaces, before, after, cursorLess)

			if first, ok := ctx.Arguments["first"].(int); ok {
				if first < 0 {
					return nil, fmt.Errorf("The `first` argument cannot be negative.")
				}
				if len(edges) > first {
					edges = edges[:first]
					hasNextPage = true
				} else {
					hasNextPage = false
				}
			}

			if last, ok := ctx.Arguments["last"].(int); ok {
				if last < 0 {
					return nil, fmt.Errorf("The `last` argument cannot be negative.")
				}
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
