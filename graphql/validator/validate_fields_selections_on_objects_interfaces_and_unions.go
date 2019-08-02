package validator

import (
	"github.com/ccbrown/apifu/graphql/ast"
	"github.com/ccbrown/apifu/graphql/schema"
)

func validateFieldsSelectionsOnObjectsInterfacesAndUnions(doc *ast.Document, s *schema.Schema, typeInfo *TypeInfo) []*Error {
	var ret []*Error
	var selectionSetType []schema.Type
	ast.Inspect(doc, func(node interface{}) bool {
		if node == nil {
			selectionSetType = selectionSetType[:len(selectionSetType)-1]
			return true
		}

		switch node := node.(type) {
		case *ast.SelectionSet:
			selectionSetType = append(selectionSetType, schema.UnwrapType(typeInfo.SelectionSetTypes[node]))
		case *ast.Field:
			name := node.Name.Name
			if name != "__typename" {
				switch parent := selectionSetType[len(selectionSetType)-1].(type) {
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
			selectionSetType = append(selectionSetType, nil)
		default:
			selectionSetType = append(selectionSetType, nil)
		}
		return true
	})
	return ret
}
