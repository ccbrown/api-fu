package apifu

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/ccbrown/api-fu/graphql"
)

// Config defines the schema and other parameters for an API.
type Config struct {
	Logger               logrus.FieldLogger
	WebSocketOriginCheck func(r *http.Request) bool

	// If given, these fields will be added to the Node interface.
	AdditionalNodeFields map[string]*graphql.FieldDefinition

	// Invoked to get nodes by their global ids.
	ResolveNodesByGlobalIds func(ctx context.Context, ids []string) ([]interface{}, error)

	// If given, Apollo persisted queries are supported by the API:
	// https://www.apollographql.com/docs/react/api/link/persisted-queries/
	PersistedQueryStorage PersistedQueryStorage

	// When calculating field costs, this is used as the default. This is typically either
	// `graphql.FieldCost{Resolver: 1}` or left as zero.
	DefaultFieldCost graphql.FieldCost

	// Execute is invoked to execute a GraphQL request. If not given, this is simply
	// graphql.Execute. You may wish to provide this to perform request logging or
	// pre/post-processing.
	Execute func(*graphql.Request, *RequestInfo) *graphql.Response

	// If given, this function is invoked when the servers receives the graphql-ws connection init
	// payload. If an error is returned, it will be sent to the client and the connection will be
	// closed. Otherwise the returned context will become associated with the connection.
	//
	// This is commonly used for authentication.
	HandleGraphQLWSInit func(ctx context.Context, parameters json.RawMessage) (context.Context, error)

	// Explicitly adds named types to the schema. This is generally only required for interface
	// implementations that aren't explicitly referenced elsewhere in the schema.
	AdditionalTypes map[string]graphql.NamedType

	// If given, these function will be executed as the schema is built. It is executed on a clone
	// of the schema and can be used to make last minute modifications to types, such as injecting
	// documentation.
	PreprocessGraphQLSchemaDefinition func(schema *graphql.SchemaDefinition) error

	initOnce      sync.Once
	nodeInterface *graphql.InterfaceType
	query         *graphql.ObjectType
	mutation      *graphql.ObjectType
	subscription  *graphql.ObjectType
}

func (cfg *Config) init() {
	cfg.initOnce.Do(func() {
		if cfg.AdditionalTypes == nil {
			cfg.AdditionalTypes = make(map[string]graphql.NamedType)
		}

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
					Cost: graphql.FieldResolverCost(1),
					Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
						// TODO: batching?
						nodes, err := ctxAPI(ctx.Context).config.ResolveNodesByGlobalIds(ctx.Context, []string{ctx.Arguments["id"].(string)})
						if err != nil || len(nodes) == 0 {
							return nil, err
						}
						return nodes[0], nil
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
					Cost: func(ctx graphql.FieldCostContext) graphql.FieldCost {
						ids, _ := ctx.Arguments["ids"].([]interface{})
						return graphql.FieldCost{
							Resolver:   1,
							Multiplier: len(ids),
						}
					},
					Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
						var ids []string
						for _, id := range ctx.Arguments["ids"].([]interface{}) {
							ids = append(ids, id.(string))
						}
						return ctxAPI(ctx.Context).config.ResolveNodesByGlobalIds(ctx.Context, ids)
					},
				},
			},
		}
	})
}

func (cfg *Config) graphqlSchemaDefinition() (*graphql.SchemaDefinition, error) {
	additionalTypes := make([]graphql.NamedType, 0, len(cfg.AdditionalTypes))
	for _, t := range cfg.AdditionalTypes {
		additionalTypes = append(additionalTypes, t)
	}
	ret := &graphql.SchemaDefinition{
		Query:           cfg.query,
		Mutation:        cfg.mutation,
		Subscription:    cfg.subscription,
		AdditionalTypes: additionalTypes,
		Directives: map[string]*graphql.DirectiveDefinition{
			"include": graphql.IncludeDirective,
			"skip":    graphql.SkipDirective,
		},
	}
	if cfg.PreprocessGraphQLSchemaDefinition != nil {
		ret = ret.Clone()
		if err := cfg.PreprocessGraphQLSchemaDefinition(ret); err != nil {
			return nil, err
		}
	}
	return ret, nil
}

func (cfg *Config) graphqlSchema() (*graphql.Schema, error) {
	def, err := cfg.graphqlSchemaDefinition()
	if err != nil {
		return nil, err
	}
	return graphql.NewSchema(def)
}

// AddNamedType adds a named type to the schema. This is generally only required for interface
// implementations that aren't explicitly referenced elsewhere in the schema.
func (cfg *Config) AddNamedType(t graphql.NamedType) {
	cfg.init()
	cfg.AdditionalTypes[t.TypeName()] = t
}

// NodeInterface returns the node interface.
func (cfg *Config) NodeInterface() *graphql.InterfaceType {
	cfg.init()
	return cfg.nodeInterface
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
//	Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
//	    if ctx.IsSubscribe {
//	        ticker := time.NewTicker(time.Second)
//	        return &apifu.SubscriptionSourceStream{
//	            EventChannel: ticker.C,
//	            Stop:         ticker.Stop,
//	        }, nil
//	    } else if ctx.Object != nil {
//	        return ctx.Object, nil
//	    } else {
//	        return nil, fmt.Errorf("Subscriptions are not supported using this protocol.")
//	    }
//	},
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
