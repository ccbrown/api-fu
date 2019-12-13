package apifu

import (
	"reflect"

	"github.com/ccbrown/api-fu/graphql"
)

func fieldValue(object interface{}, name string) interface{} {
	v := reflect.ValueOf(object)
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v.FieldByName(name).Interface()
}

func NonNullNodeID(fieldName string) *graphql.FieldDefinition {
	return &graphql.FieldDefinition{
		Type: graphql.NewNonNullType(graphql.IDType),
		Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
			cfg := ctxAPI(ctx.Context).config
			modelType := normalizeModelType(reflect.TypeOf(ctx.Object))
			nodeType := cfg.nodeTypesByModel[modelType]
			return cfg.SerializeNodeId(nodeType.Id, fieldValue(ctx.Object, fieldName)), nil
		},
	}
}

func NonNullString(fieldName string) *graphql.FieldDefinition {
	return &graphql.FieldDefinition{
		Type: graphql.NewNonNullType(graphql.StringType),
		Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
			return fieldValue(ctx.Object, fieldName), nil
		},
	}
}

func NonNullBoolean(fieldName string) *graphql.FieldDefinition {
	return &graphql.FieldDefinition{
		Type: graphql.NewNonNullType(graphql.BooleanType),
		Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
			return fieldValue(ctx.Object, fieldName), nil
		},
	}
}

func Node(nodeType *graphql.ObjectType, fieldName string) *graphql.FieldDefinition {
	return &graphql.FieldDefinition{
		Type: nodeType,
		Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
			api := ctxAPI(ctx.Context)
			nodeType, ok := api.config.nodeTypesByObjectType[nodeType]
			if !ok {
				return nil, nil
			}
			return api.resolveNodeById(ctx.Context, nodeType, fieldValue(ctx.Object, fieldName))
		},
	}
}
