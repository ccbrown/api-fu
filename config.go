package apifu

import (
	"reflect"
	"sync"

	"github.com/ccbrown/api-fu/graphql"
)

type Config struct {
	SerializeNodeId   func(typeId int, id interface{}) string
	DeserializeNodeId func(string) (typeId int, id interface{})

	initOnce          sync.Once
	nodeTypesByModel  map[reflect.Type]*Node
	nodeTypesByTypeId map[int]*Node
}

func (cfg *Config) init() {
	cfg.initOnce.Do(func() {
		cfg.nodeTypesByModel = make(map[reflect.Type]*Node)
		cfg.nodeTypesByTypeId = make(map[int]*Node)
	})
}

func (cfg *Config) graphqlSchema() (*graphql.Schema, error) {
	nodeInterface := &graphql.InterfaceType{
		Name: "Node",
		Fields: map[string]*graphql.FieldDefinition{
			"id": &graphql.FieldDefinition{
				Type: graphql.NewNonNullType(graphql.IDType),
			},
		},
	}

	query := &graphql.ObjectType{
		Name: "Query",
		Fields: map[string]*graphql.FieldDefinition{
			"node": &graphql.FieldDefinition{
				Type: nodeInterface,
				Resolve: func(*graphql.FieldContext) (interface{}, error) {
					return nil, nil
				},
			},
		},
	}

	return graphql.NewSchema(&graphql.SchemaDefinition{
		Query: query,
	})
}

func (cfg *Config) AddNode(node *Node) {
	cfg.init()

	model := normalizeModel(node.Model)
	if _, ok := cfg.nodeTypesByModel[model]; ok {
		panic("node already exists for model")
	}
	cfg.nodeTypesByModel[model] = node

	if _, ok := cfg.nodeTypesByTypeId[node.TypeId]; ok {
		panic("node already exists for type id")
	}
	cfg.nodeTypesByTypeId[node.TypeId] = node
}
