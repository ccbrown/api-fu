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
