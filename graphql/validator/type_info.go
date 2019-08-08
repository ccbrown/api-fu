package validator

import (
	"fmt"

	"github.com/ccbrown/api-fu/graphql/ast"
	"github.com/ccbrown/api-fu/graphql/schema"
)

type TypeInfo struct {
	SelectionSetTypes       map[*ast.SelectionSet]schema.NamedType
	VariableDefinitionTypes map[*ast.VariableDefinition]schema.Type
	FieldDefinitions        map[*ast.Field]*schema.FieldDefinition
	ExpectedTypes           map[ast.Value]schema.Type
	DefaultValues           map[ast.Value]interface{}
}

func schemaType(t ast.Type, s *schema.Schema) schema.Type {
	switch t := t.(type) {
	case *ast.ListType:
		if inner := schemaType(t.Type, s); inner != nil {
			return schema.NewListType(inner)
		}
	case *ast.NonNullType:
		if inner := schemaType(t.Type, s); inner != nil {
			return schema.NewNonNullType(inner)
		}
	case *ast.NamedType:
		return s.NamedType(t.Name.Name)
	default:
		panic(fmt.Sprintf("unsupported ast type: %T", t))
	}
	return nil
}

func NewTypeInfo(doc *ast.Document, s *schema.Schema) *TypeInfo {
	ret := &TypeInfo{
		SelectionSetTypes:       map[*ast.SelectionSet]schema.NamedType{},
		VariableDefinitionTypes: map[*ast.VariableDefinition]schema.Type{},
		FieldDefinitions:        map[*ast.Field]*schema.FieldDefinition{},
		ExpectedTypes:           map[ast.Value]schema.Type{},
		DefaultValues:           map[ast.Value]interface{}{},
	}

	var selectionSetScopes []schema.NamedType

	ast.Inspect(doc, func(node ast.Node) bool {
		if node == nil {
			selectionSetScopes = selectionSetScopes[:len(selectionSetScopes)-1]
			return true
		}

		var selectionSetScope schema.NamedType

		switch node := node.(type) {
		case *ast.ListValue:
			if expected, ok := ret.ExpectedTypes[node].(*schema.ListType); ok {
				for _, value := range node.Values {
					ret.ExpectedTypes[value] = expected.Type
				}
			}
		case *ast.ObjectValue:
			if expected, ok := ret.ExpectedTypes[node].(*schema.InputObjectType); ok {
				for _, field := range node.Fields {
					if expected, ok := expected.Fields[field.Name.Name]; ok {
						ret.ExpectedTypes[field.Value] = expected.Type
						if expected.DefaultValue != nil {
							if expected.DefaultValue == schema.Null {
								ret.DefaultValues[field.Value] = nil
							} else {
								ret.DefaultValues[field.Value] = expected.DefaultValue
							}
						}
					}
				}
			}
		case *ast.Directive:
			if directive := s.DirectiveDefinition(node.Name.Name); directive != nil {
				for _, arg := range node.Arguments {
					if expected, ok := directive.Arguments[arg.Name.Name]; ok {
						ret.ExpectedTypes[arg.Value] = expected.Type
						if expected.DefaultValue != nil {
							if expected.DefaultValue == schema.Null {
								ret.DefaultValues[arg.Value] = nil
							} else {
								ret.DefaultValues[arg.Value] = expected.DefaultValue
							}
						}
					}
				}
			}
		case *ast.Field:
			var field *schema.FieldDefinition
			switch parent := selectionSetScopes[len(selectionSetScopes)-1].(type) {
			case *schema.InterfaceType:
				field = parent.Fields[node.Name.Name]
			case *schema.ObjectType:
				field = parent.Fields[node.Name.Name]
			}
			if field == nil {
				break
			}

			for _, arg := range node.Arguments {
				if expected, ok := field.Arguments[arg.Name.Name]; ok {
					ret.ExpectedTypes[arg.Value] = expected.Type
					if expected.DefaultValue != nil {
						ret.DefaultValues[arg.Value] = expected.DefaultValue
					}
				}
			}

			ret.FieldDefinitions[node] = field
			selectionSetScope = schema.UnwrappedType(field.Type)
		case *ast.FragmentDefinition:
			selectionSetScope = s.NamedType(node.TypeCondition.Name.Name)
		case *ast.InlineFragment:
			if node.TypeCondition == nil {
				selectionSetScope = selectionSetScopes[len(selectionSetScopes)-1]
			} else {
				selectionSetScope = s.NamedType(node.TypeCondition.Name.Name)
			}
		case *ast.OperationDefinition:
			var t *schema.ObjectType
			if op := node.OperationType; op == nil || op.Value == "query" {
				t = s.QueryType()
			} else if op.Value == "mutation" {
				t = s.MutationType()
			} else if op.Value == "subscription" {
				t = s.SubscriptionType()
			}
			if t != nil {
				selectionSetScope = t
			}
		case *ast.SelectionSet:
			if t := selectionSetScopes[len(selectionSetScopes)-1]; t != nil {
				ret.SelectionSetTypes[node] = t
				selectionSetScope = t
			}
		case *ast.VariableDefinition:
			if t := schemaType(node.Type, s); t != nil {
				ret.VariableDefinitionTypes[node] = t
				if node.DefaultValue != nil {
					ret.ExpectedTypes[node.DefaultValue] = t
				}
			}
		}

		selectionSetScopes = append(selectionSetScopes, selectionSetScope)
		return true
	})

	return ret
}
