package schema

import (
	"math"
	"strconv"

	"github.com/ccbrown/api-fu/graphql/ast"
)

func coerceInt(v interface{}) interface{} {
	switch v := v.(type) {
	case bool:
		if v {
			return 1
		}
		return 0
	case int8:
		return int(v)
	case uint8:
		return int(v)
	case int16:
		return int(v)
	case uint16:
		return int(v)
	case int32:
		return int(v)
	case uint32:
		if v <= math.MaxInt32 {
			return int(v)
		}
	case int64:
		if v >= math.MinInt32 && v <= math.MaxInt32 {
			return int(v)
		}
	case uint64:
		if v <= math.MaxInt32 {
			return int(v)
		}
	case int:
		if v >= math.MinInt32 && v <= math.MaxInt32 {
			return int(v)
		}
	case uint:
		if v <= math.MaxInt32 {
			return int(v)
		}
	case float32:
		return coerceInt(float64(v))
	case float64:
		if n := math.Trunc(v); n == v && n >= math.MinInt32 && n <= math.MaxInt32 {
			return int(n)
		}
	}
	return nil
}

var IntType = &ScalarType{
	Name: "Int",
	LiteralCoercion: func(v ast.Value) interface{} {
		switch v := v.(type) {
		case *ast.IntValue:
			if n, err := strconv.ParseInt(v.Value, 10, 32); err == nil {
				return int(n)
			}
		}
		return nil
	},
	VariableValueCoercion: coerceInt,
	ResultCoercion:        coerceInt,
}

func coerceFloat(v interface{}) interface{} {
	switch v := v.(type) {
	case bool:
		if v {
			return 1.0
		}
		return 0.0
	case int8:
		return float64(v)
	case uint8:
		return float64(v)
	case int16:
		return float64(v)
	case uint16:
		return float64(v)
	case int32:
		return float64(v)
	case uint32:
		return float64(v)
	case int64:
		return float64(v)
	case uint64:
		return float64(v)
	case int:
		return float64(v)
	case uint:
		return float64(v)
	case float32:
		return float64(v)
	case float64:
		return v
	}
	return nil
}

var FloatType = &ScalarType{
	Name: "Float",
	LiteralCoercion: func(v ast.Value) interface{} {
		switch v := v.(type) {
		case *ast.IntValue:
			if n, err := strconv.ParseFloat(v.Value, 64); err == nil {
				return n
			}
		case *ast.FloatValue:
			if n, err := strconv.ParseFloat(v.Value, 64); err == nil {
				return n
			}
		}
		return nil
	},
	VariableValueCoercion: coerceFloat,
	ResultCoercion:        coerceFloat,
}

func coerceString(v interface{}) interface{} {
	switch v := v.(type) {
	case string:
		return v
	}
	return nil
}

var StringType = &ScalarType{
	Name: "String",
	LiteralCoercion: func(v ast.Value) interface{} {
		switch v := v.(type) {
		case *ast.StringValue:
			return v.Value
		}
		return nil
	},
	VariableValueCoercion: coerceString,
	ResultCoercion:        coerceString,
}

func coerceBoolean(v interface{}) interface{} {
	switch v := v.(type) {
	case bool:
		return v
	}
	return nil
}

var BooleanType = &ScalarType{
	Name: "Boolean",
	LiteralCoercion: func(v ast.Value) interface{} {
		switch v := v.(type) {
		case *ast.BooleanValue:
			return v.Value
		}
		return nil
	},
	VariableValueCoercion: coerceBoolean,
	ResultCoercion:        coerceBoolean,
}

var IDType = &ScalarType{
	Name: "ID",
	LiteralCoercion: func(v ast.Value) interface{} {
		switch v := v.(type) {
		case *ast.IntValue:
			if n, err := strconv.ParseInt(v.Value, 10, 0); err == nil {
				return int(n)
			}
		case *ast.StringValue:
			return v.Value
		}
		return nil
	},
	VariableValueCoercion: func(v interface{}) interface{} {
		switch v := v.(type) {
		case int:
			return v
		case float64:
			if n := int(math.Trunc(v)); float64(n) == v {
				return n
			}
		case string:
			return v
		}
		return nil
	},
	ResultCoercion: func(v interface{}) interface{} {
		switch v := v.(type) {
		case int8:
			return strconv.FormatInt(int64(v), 10)
		case uint8:
			return strconv.FormatInt(int64(v), 10)
		case int16:
			return strconv.FormatInt(int64(v), 10)
		case uint16:
			return strconv.FormatInt(int64(v), 10)
		case int32:
			return strconv.FormatInt(int64(v), 10)
		case uint32:
			return strconv.FormatInt(int64(v), 10)
		case int64:
			return strconv.FormatInt(v, 10)
		case uint64:
			if v <= math.MaxInt64 {
				return strconv.FormatInt(int64(v), 10)
			}
		case int:
			return strconv.FormatInt(int64(v), 10)
		case uint:
			if v <= math.MaxInt64 {
				return strconv.FormatInt(int64(v), 10)
			}
		case string:
			return v
		}
		return nil
	},
}

var BuiltInTypes = map[string]*ScalarType{
	"Int":     IntType,
	"Float":   FloatType,
	"String":  StringType,
	"Boolean": BooleanType,
	"ID":      IDType,
}

var SkipDirective = &DirectiveDefinition{
	Description: "The @skip directive may be provided for fields, fragment spreads, and inline fragments, and allows for conditional exclusion during execution as described by the if argument.",
	Arguments: map[string]*InputValueDefinition{
		"if": {
			Type: NewNonNullType(BooleanType),
		},
	},
	Locations: []DirectiveLocation{DirectiveLocationField, DirectiveLocationFragmentSpread, DirectiveLocationInlineFragment},
	FieldCollectionFilter: func(arguments map[string]interface{}) bool {
		return !arguments["if"].(bool)
	},
}

var IncludeDirective = &DirectiveDefinition{
	Description: "The @include directive may be provided for fields, fragment spreads, and inline fragments, and allows for conditional inclusion during execution as described by the if argument.",
	Arguments: map[string]*InputValueDefinition{
		"if": {
			Type: NewNonNullType(BooleanType),
		},
	},
	Locations: []DirectiveLocation{DirectiveLocationField, DirectiveLocationFragmentSpread, DirectiveLocationInlineFragment},
	FieldCollectionFilter: func(arguments map[string]interface{}) bool {
		return arguments["if"].(bool)
	},
}
