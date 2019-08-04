package schema

import (
	"strconv"

	"github.com/ccbrown/api-fu/graphql/ast"
)

var IntType = &ScalarType{
	Name: "Int",
	CoerceLiteral: func(v ast.Value) interface{} {
		switch v := v.(type) {
		case *ast.IntValue:
			if n, err := strconv.ParseInt(v.Value, 10, 32); err == nil {
				return int(n)
			}
		}
		return nil
	},
}

var FloatType = &ScalarType{
	Name: "Float",
	CoerceLiteral: func(v ast.Value) interface{} {
		switch v := v.(type) {
		case *ast.IntValue:
			if n, err := strconv.ParseFloat(v.Value, 64); err == nil {
				return int(n)
			}
		case *ast.FloatValue:
			if n, err := strconv.ParseFloat(v.Value, 64); err == nil {
				return int(n)
			}
		}
		return nil
	},
}

var StringType = &ScalarType{
	Name: "String",
	CoerceLiteral: func(v ast.Value) interface{} {
		switch v := v.(type) {
		case *ast.StringValue:
			return v.Value
		}
		return nil
	},
}

var BooleanType = &ScalarType{
	Name: "Boolean",
	CoerceLiteral: func(v ast.Value) interface{} {
		switch v := v.(type) {
		case *ast.BooleanValue:
			return v.Value
		}
		return nil
	},
}

var IDType = &ScalarType{
	Name: "ID",
	CoerceLiteral: func(v ast.Value) interface{} {
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
}

var builtins = map[string]*ScalarType{
	"Int":     IntType,
	"Float":   FloatType,
	"String":  StringType,
	"Boolean": BooleanType,
	"ID":      IDType,
}
