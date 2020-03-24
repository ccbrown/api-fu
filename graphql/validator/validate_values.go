package validator

import (
	"fmt"

	"github.com/ccbrown/api-fu/graphql/ast"
	"github.com/ccbrown/api-fu/graphql/schema"
)

func validateValues(doc *ast.Document, s *schema.Schema, typeInfo *TypeInfo) []*Error {
	var ret []*Error

	ast.Inspect(doc, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.Variable:
			// variable types are validated by variable validation rules
		case ast.Value:
			if expected, ok := typeInfo.ExpectedTypes[node]; ok {
				ret = append(ret, validateCoercion(node, expected, true)...)
			} else {
				ret = append(ret, newSecondaryError(node, "no type info for value"))
			}
			return false
		}
		return true
	})

	return ret
}

func validateCoercion(from ast.Value, to schema.Type, allowItemToListCoercion bool) []*Error {
	var ret []*Error

	if _, ok := from.(*ast.Variable); ok {
		// variable types are validated by variable validation rules
		return ret
	}

	if ast.IsNullValue(from) {
		if schema.IsNonNullType(to) {
			ret = append(ret, newError(from, "cannot coerce null to non-null type"))
		}
		return ret
	}

	switch to := to.(type) {
	case *schema.ScalarType:
		if to.LiteralCoercion != nil && to.LiteralCoercion(from) == nil {
			ret = append(ret, newError(from, "cannot coerce to %v", to))
		}
	case *schema.ListType:
		if fromList, ok := from.(*ast.ListValue); ok {
			for _, value := range fromList.Values {
				if err := validateCoercion(value, to.Type, false); err != nil {
					return err
				}
			}
			return ret
		} else if allowItemToListCoercion {
			return validateCoercion(from, to.Type, true)
		}
		ret = append(ret, newError(from, "cannot coerce to %v", to))
	case *schema.InputObjectType:
		if from, ok := from.(*ast.ObjectValue); ok {
			fieldsByName := map[string]*ast.ObjectField{}
			for _, field := range from.Fields {
				if _, ok := fieldsByName[field.Name.Name]; ok {
					ret = append(ret, newError(field, "duplicate field"))
				}
				fieldsByName[field.Name.Name] = field

				if def, ok := to.Fields[field.Name.Name]; ok {
					if err := validateCoercion(field.Value, def.Type, true); err != nil {
						return err
					}
				} else {
					ret = append(ret, newError(field, "field does not exist on %v", to.Name))
				}
			}

			for name, field := range to.Fields {
				if schema.IsNonNullType(field.Type) && field.DefaultValue == nil {
					if _, ok := fieldsByName[name]; !ok {
						ret = append(ret, newError(from, "the %v field is required", name))
					}
				}
			}
			return ret
		}
		ret = append(ret, newError(from, "cannot coerce to %v", to))
	case *schema.EnumType:
		if _, err := to.CoerceLiteral(from); err != nil {
			ret = append(ret, newError(from, "cannot coerce to %v", to))
		}
	case *schema.NonNullType:
		return validateCoercion(from, to.Type, allowItemToListCoercion)
	default:
		panic(fmt.Sprintf("unsupported input coercion type: %T", to))
	}
	return ret
}
