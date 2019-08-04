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
		expectedType := typeInfo.ExpectedTypes[node]

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
						if given, ok := fieldsByName[name]; !ok || ast.IsNullValue(given.Value) {
							ret = append(ret, newError("the %v field is required", name))
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

	return ret
}
