package apifu

import (
	"time"

	"github.com/ccbrown/api-fu/graphql"
	"github.com/ccbrown/api-fu/graphql/ast"
)

func parseDateTime(v interface{}) interface{} {
	switch v := v.(type) {
	case []byte:
		t := time.Time{}
		if err := t.UnmarshalText(v); err == nil {
			return t
		}
		return nil
	case string:
		return parseDateTime([]byte(v))
	}
	return nil
}

var DateTimeType = &graphql.ScalarType{
	Name:        "DateTime",
	Description: "DateTime represents an RFC-3339 datetime.",
	LiteralCoercion: func(v ast.Value) interface{} {
		switch v := v.(type) {
		case *ast.StringValue:
			return parseDateTime(v.Value)
		}
		return nil
	},
	VariableValueCoercion: parseDateTime,
	ResultCoercion: func(v interface{}) interface{} {
		switch v := v.(type) {
		case time.Time:
			if b, err := v.MarshalText(); err == nil {
				return string(b)
			}
		}
		return nil
	},
}

func NonNullDateTime(fieldName string) *graphql.FieldDefinition {
	return &graphql.FieldDefinition{
		Type: graphql.NewNonNullType(DateTimeType),
		Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
			return fieldValue(ctx.Object, fieldName), nil
		},
	}
}
