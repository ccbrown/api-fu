package validator

import (
	"github.com/ccbrown/apifu/graphql/ast"
	"github.com/ccbrown/apifu/graphql/schema"
)

type TypeInfo struct {
	SelectionSetTypes map[*ast.SelectionSet]schema.Type
	FieldTypes        map[*ast.Field]schema.Type
}

func NewTypeInfo(doc *ast.Document, s *schema.Schema) *TypeInfo {
	ret := &TypeInfo{
		SelectionSetTypes: map[*ast.SelectionSet]schema.Type{},
		FieldTypes:        map[*ast.Field]schema.Type{},
	}

	var selectionSetScopes []schema.Type

	ast.Inspect(doc, func(node interface{}) bool {
		if node == nil {
			selectionSetScopes = selectionSetScopes[:len(selectionSetScopes)-1]
			return true
		}

		switch node := node.(type) {
		case *ast.Field:
			var fieldType schema.Type
			switch parent := selectionSetScopes[len(selectionSetScopes)-1].(type) {
			case *schema.InterfaceType:
				if f := parent.Fields[node.Name.Name]; f != nil {
					fieldType = f.Type
				}
			case *schema.ObjectType:
				if f := parent.Fields[node.Name.Name]; f != nil {
					fieldType = f.Type
				}
			}
			if fieldType != nil {
				ret.FieldTypes[node] = fieldType
			}
			selectionSetScopes = append(selectionSetScopes, fieldType)
		case *ast.FragmentDefinition:
			selectionSetScopes = append(selectionSetScopes, s.NamedType(node.TypeCondition.Name.Name))
		case *ast.InlineFragment:
			if node.TypeCondition == nil {
				selectionSetScopes = append(selectionSetScopes, selectionSetScopes[len(selectionSetScopes)-1])
			} else {
				selectionSetScopes = append(selectionSetScopes, s.NamedType(node.TypeCondition.Name.Name))
			}
		case *ast.OperationDefinition:
			if op := node.OperationType; op == nil || *op == ast.OperationTypeQuery {
				selectionSetScopes = append(selectionSetScopes, s.QueryType())
			} else if *op == ast.OperationTypeMutation {
				selectionSetScopes = append(selectionSetScopes, s.MutationType())
			} else if *op == ast.OperationTypeSubscription {
				selectionSetScopes = append(selectionSetScopes, s.SubscriptionType())
			} else {
				selectionSetScopes = append(selectionSetScopes, nil)
			}
		case *ast.SelectionSet:
			t := selectionSetScopes[len(selectionSetScopes)-1]
			ret.SelectionSetTypes[node] = t
			selectionSetScopes = append(selectionSetScopes, t)
		default:
			selectionSetScopes = append(selectionSetScopes, nil)
		}
		return true
	})

	return ret
}
