package apifu

import (
	"context"
	"reflect"

	"github.com/ccbrown/api-fu/graphql"
)

type NodeType struct {
	// Id should be an integer that uniquely identifies the node type. Once set, it should never
	// change and no other nodes should ever use the same id.
	Id int

	Name string

	Model reflect.Type

	// GetByIds should be a function that accepts a context and slice of ids and returns a slice of
	// models.
	GetByIds func(ctx context.Context, ids interface{}) (models interface{}, err error)

	Fields map[string]*graphql.FieldDefinition
}

func NonNullID(modelField string) *graphql.FieldDefinition {
	return &graphql.FieldDefinition{
		Type: graphql.NewNonNullType(graphql.IDType),
	}
}
func NonNullString(modelField string) *graphql.FieldDefinition {
	return &graphql.FieldDefinition{
		Type: graphql.NewNonNullType(graphql.StringType),
	}
}
