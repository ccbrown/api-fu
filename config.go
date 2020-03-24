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
	nodeObjectTypesByName map[string]*graphql.ObjectType
	nodeTypesByModel      map[reflect.Type]*NodeType
	nodeTypesById         map[int]*NodeType
	nodeTypesByObjectType map[*graphql.ObjectType]*NodeType
	nodeInterface         *graphql.InterfaceType
	query                 *graphql.ObjectType
	mutation              *graphql.ObjectType
	subscription          *graphql.ObjectType
	additionalTypes       []graphql.NamedType
}

func (cfg *Config) init() {
	cfg.initOnce.Do(func() {
		cfg.nodeObjectTypesByName = make(map[string]*graphql.ObjectType)
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
				"nodes": &graphql.FieldDefinition{
					Type:        graphql.NewListType(cfg.nodeInterface),
					Description: "Gets nodes for multiple ids. Non-existent nodes are not returned and the order of the returned nodes is arbitrary, so clients should check the ids of the returned nodes.",
					Arguments: map[string]*graphql.InputValueDefinition{
						"ids": &graphql.InputValueDefinition{
							Type: graphql.NewNonNullType(graphql.NewListType(graphql.NewNonNullType(graphql.IDType))),
						},
					},
					Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
						var ids []string
						for _, id := range ctx.Arguments["ids"].([]interface{}) {
							ids = append(ids, id.(string))
						}
						return ctxAPI(ctx.Context).resolveNodesByGlobalIds(ctx.Context, ids)
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
		Subscription:    cfg.subscription,
		AdditionalTypes: cfg.additionalTypes,
		Directives: map[string]*graphql.DirectiveDefinition{
			"include": graphql.IncludeDirective,
			"skip":    graphql.SkipDirective,
		},
	})
}

func (cfg *Config) NodeObjectType(name string) *graphql.ObjectType {
	return cfg.nodeObjectTypesByName[name]
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
	cfg.nodeObjectTypesByName[t.Name] = objectType

	return objectType
}

// Adds a named type to the schema. This is generally only required for interface implementations
// that aren't explicitly referenced elsewhere in the schema.
func (cfg *Config) AddNamedType(t graphql.NamedType) {
	cfg.init()
	cfg.additionalTypes = append(cfg.additionalTypes, t)
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

// When a subscription is started, your resolver will be invoked with ctx.IsSubscribe set to true.
// When this happens, you should return a pointer to a SubscriptionSourceStream (or an error). For
// example:
//
//     Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
//         if ctx.IsSubscribe {
//             ticker := time.NewTicker(time.Second)
//             return &apifu.SubscriptionSourceStream{
//                 EventChannel: ticker.C,
//                 Stop:         ticker.Stop,
//             }, nil
//         } else if ctx.Object != nil {
//             return ctx.Object, nil
//         } else {
//             return nil, fmt.Errorf("Subscriptions are not supported using this protocol.")
//         }
//     },
func (cfg *Config) AddSubscription(name string, def *graphql.FieldDefinition) {
	cfg.init()

	if cfg.subscription == nil {
		cfg.subscription = &graphql.ObjectType{
			Name:   "Subscription",
			Fields: map[string]*graphql.FieldDefinition{},
		}
	}

	if _, ok := cfg.subscription.Fields[name]; ok {
		panic("a subscription with that name already exists")
	}

	cfg.subscription.Fields[name] = def
}

func (cfg *Config) AddQueryField(name string, def *graphql.FieldDefinition) {
	cfg.init()

	if _, ok := cfg.query.Fields[name]; ok {
		panic("a field with that name already exists")
	}

	cfg.query.Fields[name] = def
}
