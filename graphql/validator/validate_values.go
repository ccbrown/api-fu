package validator

import (
	"github.com/ccbrown/api-fu/graphql/ast"
	"github.com/ccbrown/api-fu/graphql/schema"
)

func validateValues(doc *ast.Document, s *schema.Schema, typeInfo *TypeInfo) []*Error {
	var ret []*Error

	parentTypes := []schema.Type{nil}
	ast.Inspect(doc, func(node interface{}) bool {
		if node == nil {
			parentTypes = parentTypes[:len(parentTypes)-1]
			return true
		}

		parentType := parentTypes[len(parentTypes)-1]

		var expectedType schema.Type
		if value, ok := node.(ast.Value); ok {
			expectedType = typeInfo.ExpectedTypes[value]
		}

		switch node := node.(type) {
		case *ast.ObjectValue:
			fieldsByName := map[string]*ast.ObjectField{}
			for _, field := range node.Fields {
				if _, ok := fieldsByName[field.Name.Name]; ok {
					ret = append(ret, newError("duplicate field"))
				}
				fieldsByName[field.Name.Name] = field
			}

			if def, ok := expectedType.(*schema.InputObjectType); ok {
				for name, field := range def.Fields {
					if schema.IsNonNullType(field.Type) && field.DefaultValue == nil {
						if given, ok := fieldsByName[name]; !ok {
							ret = append(ret, newError("the %v field is required", name))
						} else if ast.IsNullValue(given.Value) {
							// primarily checked during value coercion validation
							ret = append(ret, newSecondaryError("the %v field cannot be null", name))
						}
					}
				}
			} else {
				ret = append(ret, newSecondaryError("no type info for input object"))
			}
		case *ast.ObjectField:
			if parent, ok := parentType.(*schema.InputObjectType); ok {
				if _, ok := parent.Fields[node.Name.Name]; !ok {
					ret = append(ret, newError("field does not exist on %v", parent.Name))
				}
			}
		}

		parentTypes = append(parentTypes, expectedType)
		return true
	})

	ast.Inspect(doc, func(node interface{}) bool {
		switch node := node.(type) {
		case *ast.Variable:
			// variable types are validated by variable validation rules
			return false
		case ast.Value:
			if expected, ok := typeInfo.ExpectedTypes[node]; ok {
				if err := validateShallowCoercion(node, expected); err != nil {
					ret = append(ret, err)
					return false
				}
			} else {
				ret = append(ret, newSecondaryError("no type info for value"))
				return false
			}
		}
		return true
	})

	return ret
}

func validateShallowCoercion(from ast.Value, to schema.Type) *Error {
	if ast.IsNullValue(from) {
		if schema.IsNonNullType(to) {
			return newError("cannot coerce null to non-null type")
		}
		return nil
	}

	switch to := to.(type) {
	case *schema.ScalarType:
		if to.CoerceLiteral(from) != nil {
			return nil
		}
		return newError("cannot coerce to %v", to)
	case *schema.ListType:
		if _, ok := from.(*ast.ListValue); ok {
			return nil
		}
		return newError("cannot coerce to %v", to)
	case *schema.InputObjectType:
		if _, ok := from.(*ast.ObjectValue); ok {
			return nil
		}
		return newError("cannot coerce to %v", to)
	case *schema.EnumType:
		if from, ok := from.(*ast.EnumValue); ok {
			if _, ok := to.Values[from.Value]; !ok {
				return newError("undefined enum value for %v", to)
			}
			return nil
		}
		return newError("cannot coerce to %v", to)
	case *schema.NonNullType:
		return validateShallowCoercion(from, to.Type)
	}

	panic("unsupported input coercion type")
}
