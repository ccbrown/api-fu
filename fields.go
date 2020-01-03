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

func OwnID(fieldName string) *graphql.FieldDefinition {
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

func NonNullNodeID(modelType reflect.Type, fieldName string) *graphql.FieldDefinition {
	return &graphql.FieldDefinition{
		Type: graphql.NewNonNullType(graphql.IDType),
		Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
			cfg := ctxAPI(ctx.Context).config
			modelType = normalizeModelType(modelType)
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

// Returns a field that resolves to a string if the field's value is non-empty. Otherwise, the field
// resolves to nil.
func NonEmptyString(fieldName string) *graphql.FieldDefinition {
	return &graphql.FieldDefinition{
		Type: graphql.StringType,
		Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
			if s := fieldValue(ctx.Object, fieldName); s != "" {
				return s, nil
			}
			return nil, nil
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

func NonNullInt(fieldName string) *graphql.FieldDefinition {
	return &graphql.FieldDefinition{
		Type: graphql.NewNonNullType(graphql.IntType),
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
