package apifu

import (
	"reflect"
	"sync"

	"github.com/ccbrown/api-fu/graphql"
)

type Config struct {
	SerializeNodeId   func(typeId int, id interface{}) string
	DeserializeNodeId func(string) (typeId int, id interface{})

	initOnce         sync.Once
	nodeTypesByModel map[reflect.Type]*NodeType
	nodeTypesById    map[int]*NodeType
	nodeInterface    *graphql.InterfaceType
	query            *graphql.ObjectType
	mutation         *graphql.ObjectType
	additionalTypes  []graphql.NamedType
}

func (cfg *Config) init() {
	cfg.initOnce.Do(func() {
		cfg.nodeTypesByModel = make(map[reflect.Type]*NodeType)
		cfg.nodeTypesById = make(map[int]*NodeType)

		cfg.nodeInterface = &graphql.InterfaceType{
			Name: "Node",
			Fields: map[string]*graphql.FieldDefinition{
				"id": &graphql.FieldDefinition{
					Type: graphql.NewNonNullType(graphql.IDType),
				},
			},
		}

		cfg.query = &graphql.ObjectType{
			Name: "Query",
			Fields: map[string]*graphql.FieldDefinition{
				"node": &graphql.FieldDefinition{
					Type: cfg.nodeInterface,
					Resolve: func(*graphql.FieldContext) (interface{}, error) {
						return nil, nil
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
	})
}

func (cfg *Config) AddNodeType(t *NodeType) *graphql.ObjectType {
	cfg.init()

	model := normalizeModel(t.Model)
	if _, ok := cfg.nodeTypesByModel[model]; ok {
		panic("node type already exists for model")
	}
	cfg.nodeTypesByModel[model] = t

	if _, ok := cfg.nodeTypesById[t.Id]; ok {
		panic("node type already exists for type id")
	}
	cfg.nodeTypesById[t.Id] = t

	fields := make(map[string]*graphql.FieldDefinition, len(t.Fields)+2)
	for k, v := range t.Fields {
		fields[k] = v
	}
	fields["id"] = &graphql.FieldDefinition{
		Type: graphql.NewNonNullType(graphql.IDType),
		Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
			// TODO
			return "foo", nil
		},
	}

	objectType := &graphql.ObjectType{
		Name:                  t.Name,
		Fields:                fields,
		ImplementedInterfaces: []*graphql.InterfaceType{cfg.nodeInterface},
	}
	cfg.additionalTypes = append(cfg.additionalTypes, objectType)

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
