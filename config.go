package apifu

import (
	"net/http"
	"reflect"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/ccbrown/api-fu/graphql"
)

type Config struct {
	Logger               logrus.FieldLogger
	WebSocketOriginCheck func(r *http.Request) bool
	SerializeNodeId      func(typeId int, id interface{}) string
	DeserializeNodeId    func(string) (typeId int, id interface{})
	AdditionalNodeFields map[string]*graphql.FieldDefinition

	initOnce              sync.Once
	nodeTypesByModel      map[reflect.Type]*NodeType
	nodeTypesById         map[int]*NodeType
	nodeTypesByObjectType map[*graphql.ObjectType]*NodeType
	nodeInterface         *graphql.InterfaceType
	query                 *graphql.ObjectType
	mutation              *graphql.ObjectType
	additionalTypes       []graphql.NamedType
}

func (cfg *Config) init() {
	cfg.initOnce.Do(func() {
		cfg.nodeTypesByModel = make(map[reflect.Type]*NodeType)
		cfg.nodeTypesById = make(map[int]*NodeType)
		cfg.nodeTypesByObjectType = make(map[*graphql.ObjectType]*NodeType)

		cfg.nodeInterface = &graphql.InterfaceType{
			Name: "Node",
			Fields: map[string]*graphql.FieldDefinition{
				"id": &graphql.FieldDefinition{
					Type: graphql.NewNonNullType(graphql.IDType),
				},
			},
		}
		for k, v := range cfg.AdditionalNodeFields {
			cfg.nodeInterface.Fields[k] = v
		}

		cfg.query = &graphql.ObjectType{
			Name: "Query",
			Fields: map[string]*graphql.FieldDefinition{
				"node": &graphql.FieldDefinition{
					Type: cfg.nodeInterface,
					Arguments: map[string]*graphql.InputValueDefinition{
						"id": &graphql.InputValueDefinition{
							Type: graphql.NewNonNullType(graphql.IDType),
						},
					},
					Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
						return ctxAPI(ctx.Context).resolveNodeByGlobalId(ctx.Context, ctx.Arguments["id"].(string))
					},
				},
			},
		}
	})
}

func (cfg *Config) graphqlSchema() (*graphql.Schema, error) {
	return graphql.NewSchema(&graphql.SchemaDefinition{
		Query:           cfg.query,
		Mutation:        cfg.mutation,
		AdditionalTypes: cfg.additionalTypes,
		Directives: map[string]*graphql.DirectiveDefinition{
			"include": graphql.IncludeDirective,
			"skip":    graphql.SkipDirective,
		},
	})
}

func (cfg *Config) AddNodeType(t *NodeType) *graphql.ObjectType {
	cfg.init()

	model := normalizeModelType(t.Model)
	if _, ok := cfg.nodeTypesByModel[model]; ok {
		panic("node type already exists for model")
	}
	cfg.nodeTypesByModel[model] = t

	if _, ok := cfg.nodeTypesById[t.Id]; ok {
		panic("node type already exists for type id")
	}
	cfg.nodeTypesById[t.Id] = t

	objectType := &graphql.ObjectType{
		Name:                  t.Name,
		Fields:                t.Fields,
		ImplementedInterfaces: []*graphql.InterfaceType{cfg.nodeInterface},
		IsTypeOf: func(v interface{}) bool {
			return normalizeModelType(reflect.TypeOf(v)) == model
		},
	}
	cfg.additionalTypes = append(cfg.additionalTypes, objectType)
	cfg.nodeTypesByObjectType[objectType] = t

	return objectType
}

func (cfg *Config) AddMutation(name string, def *graphql.FieldDefinition) {
	cfg.init()

	if cfg.mutation == nil {
		cfg.mutation = &graphql.ObjectType{
			Name:   "Mutation",
			Fields: map[string]*graphql.FieldDefinition{},
		}
	}

	if _, ok := cfg.mutation.Fields[name]; ok {
		panic("a mutation with that name already exists")
	}

	cfg.mutation.Fields[name] = def
}

func (cfg *Config) AddQueryField(name string, def *graphql.FieldDefinition) {
	cfg.init()

	if _, ok := cfg.query.Fields[name]; ok {
		panic("a field with that name already exists")
	}

	cfg.query.Fields[name] = def
}
