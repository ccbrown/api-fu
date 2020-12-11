package apifu

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/ccbrown/api-fu/graphql"
)

// Config defines the schema and other parameters for an API.
type Config struct {
	Logger               logrus.FieldLogger
	WebSocketOriginCheck func(r *http.Request) bool
	SerializeNodeId      func(typeId int, id interface{}) string
	DeserializeNodeId    func(string) (typeId int, id interface{})
	AdditionalNodeFields map[string]*graphql.FieldDefinition

	// If given, Apollo persisted queries are supported by the API:
	// https://www.apollographql.com/docs/react/api/link/persisted-queries/
	PersistedQueryStorage PersistedQueryStorage

	// Execute is invoked to execute a GraphQL request. If not given, this is simply
	// graphql.Execute. You may wish to provide this to perform request logging or
	// pre/post-processing.
	Execute func(*graphql.Request) *graphql.Response

	// If given, this function is invoked when the servers receives the graphql-ws connection init
	// payload. If an error is returned, it will be sent to the client and the connection will be
	// closed. Otherwise the returned context will become associated with the connection.
	//
	// This is commonly used for authentication.
	HandleGraphQLWSInit func(ctx context.Context, parameters json.RawMessage) (context.Context, error)

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
				"id": {
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
				"node": {
					Type: cfg.nodeInterface,
					Arguments: map[string]*graphql.InputValueDefinition{
						"id": {
							Type: graphql.NewNonNullType(graphql.IDType),
						},
					},
					Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
						// TODO: batching?
						return ctxAPI(ctx.Context).resolveNodeByGlobalId(ctx.Context, ctx.Arguments["id"].(string))
					},
				},
				"nodes": {
					Type:        graphql.NewListType(cfg.nodeInterface),
					Description: "Gets nodes for multiple ids. Non-existent nodes are not returned and the order of the returned nodes is arbitrary, so clients should check their ids.",
					Arguments: map[string]*graphql.InputValueDefinition{
						"ids": {
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

// NodeObjectType returns the object type for a node type previously added via AddNodeType.
func (cfg *Config) NodeObjectType(name string) *graphql.ObjectType {
	return cfg.nodeObjectTypesByName[name]
}

// AddNodeType registers the given node type and returned the object type created for the node.
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

// AddNamedType adds a named type to the schema. This is generally only required for interface
// implementations that aren't explicitly referenced elsewhere in the schema.
func (cfg *Config) AddNamedType(t graphql.NamedType) {
	cfg.init()
	cfg.additionalTypes = append(cfg.additionalTypes, t)
}

// MutationType returns the root mutation type.
func (cfg *Config) MutationType() *graphql.ObjectType {
	cfg.init()

	if cfg.mutation == nil {
		cfg.mutation = &graphql.ObjectType{
			Name:   "Mutation",
			Fields: map[string]*graphql.FieldDefinition{},
		}
	}

	return cfg.mutation
}

// AddMutation adds a mutation to your schema.
func (cfg *Config) AddMutation(name string, def *graphql.FieldDefinition) {
	t := cfg.MutationType()

	if _, ok := t.Fields[name]; ok {
		panic("a mutation with that name already exists")
	}

	t.Fields[name] = def
}

// AddSubscription adds a subscription operation to your schema.
//
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

// QueryType returns the root query type.
func (cfg *Config) QueryType() *graphql.ObjectType {
	cfg.init()
	return cfg.query
}

// AddQueryField adds a field to your schema's query object.
func (cfg *Config) AddQueryField(name string, def *graphql.FieldDefinition) {
	t := cfg.QueryType()

	if _, ok := t.Fields[name]; ok {
		panic("a field with that name already exists")
	}

	t.Fields[name] = def
}
