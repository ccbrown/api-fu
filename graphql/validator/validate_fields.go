package validator

import (
	"github.com/ccbrown/apifu/graphql/ast"
	"github.com/ccbrown/apifu/graphql/schema"
)

func validateFields(doc *ast.Document, s *schema.Schema, typeInfo *TypeInfo) []*Error {
	var ret []*Error
	var selectionSetTypes []schema.Type
	ast.Inspect(doc, func(node interface{}) bool {
		if node == nil {
			selectionSetTypes = selectionSetTypes[:len(selectionSetTypes)-1]
			return true
		}

		var selectionSetType schema.Type

		switch node := node.(type) {
		case *ast.SelectionSet:
			selectionSetType = schema.UnwrappedType(typeInfo.SelectionSetTypes[node])
		case *ast.Field:
			if def := typeInfo.FieldDefinitions[node]; def != nil {
				switch schema.UnwrappedType(def.Type).(type) {
				case *schema.ObjectType, *schema.InterfaceType, *schema.UnionType:
					if node.SelectionSet == nil {
						ret = append(ret, NewError("%v field must have a subselection", node.Name.Name))
					}
				default:
					if node.SelectionSet != nil {
						ret = append(ret, NewError("%v field cannot have a subselection", node.Name.Name))
					}
				}
			}

			name := node.Name.Name
			if name != "__typename" {
				switch parent := selectionSetTypes[len(selectionSetTypes)-1].(type) {
				case *schema.ObjectType:
					if _, ok := parent.Fields[name]; !ok {
						ret = append(ret, NewError("field %v does not exist on %v object", name, parent.Name))
					}
				case *schema.InterfaceType:
					if _, ok := parent.Fields[name]; !ok {
						ret = append(ret, NewError("field %v does not exist on %v interface", name, parent.Name))
					}
				case *schema.UnionType:
					ret = append(ret, NewError("field %v does not exist on %v union", name, parent.Name))
				}
			}
		}

		selectionSetTypes = append(selectionSetTypes, selectionSetType)
		return true
	})
	return ret
}
