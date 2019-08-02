package validator

import (
	"github.com/ccbrown/apifu/graphql/ast"
	"github.com/ccbrown/apifu/graphql/schema"
)

func validateFieldsLeafFieldSelections(doc *ast.Document, s *schema.Schema, typeInfo *TypeInfo) []*Error {
	var ret []*Error
	ast.Inspect(doc, func(node interface{}) bool {
		switch node := node.(type) {
		case *ast.Field:
			switch schema.UnwrapType(typeInfo.FieldTypes[node]).(type) {
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
		return true
	})
	return ret
}
