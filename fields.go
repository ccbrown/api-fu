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

// OwnID returns a field that resolves to an ID for the current Object.
func OwnID(fieldName string) *graphql.FieldDefinition {
	return &graphql.FieldDefinition{
		Type: graphql.NewNonNullType(graphql.IDType),
		Cost: graphql.FieldResolverCost(0),
		Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
			cfg := ctxAPI(ctx.Context).config
			modelType := normalizeModelType(reflect.TypeOf(ctx.Object))
			nodeType := cfg.nodeTypesByModel[modelType]
			return cfg.SerializeNodeId(nodeType.Id, fieldValue(ctx.Object, fieldName)), nil
		},
	}
}

// NonNullNodeID returns a field that resolves to an ID for an object of the given type.
func NonNullNodeID(modelType reflect.Type, fieldName string) *graphql.FieldDefinition {
	return &graphql.FieldDefinition{
		Type: graphql.NewNonNullType(graphql.IDType),
		Cost: graphql.FieldResolverCost(0),
		Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
			cfg := ctxAPI(ctx.Context).config
			modelType = normalizeModelType(modelType)
			nodeType := cfg.nodeTypesByModel[modelType]
			return cfg.SerializeNodeId(nodeType.Id, fieldValue(ctx.Object, fieldName)), nil
		},
	}
}

// NonEmptyString returns a field that resolves to a string if the field's value is non-empty.
// Otherwise, the field resolves to nil.
func NonEmptyString(fieldName string) *graphql.FieldDefinition {
	return &graphql.FieldDefinition{
		Type: graphql.StringType,
		Cost: graphql.FieldResolverCost(0),
		Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
			if s := fieldValue(ctx.Object, fieldName); s != "" {
				return s, nil
			}
			return nil, nil
		},
	}
}

// NonNull returns a non-null field that resolves to the given type.
func NonNull(t graphql.Type, fieldName string) *graphql.FieldDefinition {
	return &graphql.FieldDefinition{
		Type: graphql.NewNonNullType(t),
		Cost: graphql.FieldResolverCost(0),
		Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
			return fieldValue(ctx.Object, fieldName), nil
		},
	}
}

// Node returns a field that resolves to the node of the given type, whose id is the value of the
// specified field.
func Node(nodeType *graphql.ObjectType, idFieldName string) *graphql.FieldDefinition {
	return &graphql.FieldDefinition{
		Type: nodeType,
		Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
			api := ctxAPI(ctx.Context)
			nodeType, ok := api.config.nodeTypesByObjectType[nodeType]
			if !ok {
				return nil, nil
			}
			return api.resolveNodeById(ctx.Context, nodeType, fieldValue(ctx.Object, idFieldName))
		},
	}
}
