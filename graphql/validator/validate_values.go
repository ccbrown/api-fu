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
		case ast.Value:
			if expected, ok := typeInfo.ExpectedTypes[node]; ok {
				if err := validateCoercion(node, expected, true); err != nil {
					ret = append(ret, err)
				}
			} else {
				ret = append(ret, newSecondaryError("no type info for value"))
			}
			return false
		}
		return true
	})

	return ret
}

func validateCoercion(from ast.Value, to schema.Type, allowItemToListCoercion bool) *Error {
	if _, ok := from.(*ast.Variable); ok {
		// variable types are validated by variable validation rules
		return nil
	}

	if ast.IsNullValue(from) {
		if schema.IsNonNullType(to) {
			return newError("cannot coerce null to non-null type")
		}
		return nil
	}

	switch to := to.(type) {
	case *schema.ScalarType:
		if to.LiteralCoercion(from) != nil {
			return nil
		}
		return newError("cannot coerce to %v", to)
	case *schema.ListType:
		if fromList, ok := from.(*ast.ListValue); ok {
			for _, value := range fromList.Values {
				if err := validateCoercion(value, to.Type, false); err != nil {
					return err
				}
			}
			return nil
		} else if allowItemToListCoercion {
			return validateCoercion(from, to.Type, true)
		}
		return newError("cannot coerce to %v", to)
	case *schema.InputObjectType:
		if from, ok := from.(*ast.ObjectValue); ok {
			for _, field := range from.Fields {
				if def, ok := to.Fields[field.Name.Name]; ok {
					if err := validateCoercion(field.Value, def.Type, true); err != nil {
						return err
					}
				}
			}
			return nil
		}
		return newError("cannot coerce to %v", to)
	case *schema.EnumType:
		if _, err := to.CoerceLiteral(from); err == nil {
			return nil
		}
		return newError("cannot coerce to %v", to)
	case *schema.NonNullType:
		return validateCoercion(from, to.Type, allowItemToListCoercion)
	}

	panic("unsupported input coercion type")
}
