package validator

import (
	"fmt"

	"github.com/ccbrown/apifu/graphql/ast"
	"github.com/ccbrown/apifu/graphql/schema"
)

type TypeInfo struct {
	SelectionSetTypes       map[*ast.SelectionSet]schema.Type
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
		return nil
	case *ast.NonNullType:
		if inner := schemaType(t.Type, s); inner != nil {
			return schema.NewNonNullType(inner)
		}
		return nil
	case *ast.NamedType:
		return s.NamedType(t.Name.Name)
	default:
		panic(fmt.Sprintf("unsupported ast type: %T", t))
	}
}

func newTypeInfo(doc *ast.Document, s *schema.Schema) *TypeInfo {
	ret := &TypeInfo{
		SelectionSetTypes:       map[*ast.SelectionSet]schema.Type{},
		VariableDefinitionTypes: map[*ast.VariableDefinition]schema.Type{},
		FieldDefinitions:        map[*ast.Field]*schema.FieldDefinition{},
		ExpectedTypes:           map[ast.Value]schema.Type{},
		DefaultValues:           map[ast.Value]interface{}{},
	}

	var selectionSetScopes []schema.Type

	ast.Inspect(doc, func(node interface{}) bool {
		if node == nil {
			selectionSetScopes = selectionSetScopes[:len(selectionSetScopes)-1]
			return true
		}

		var selectionSetScope schema.Type

		switch node := node.(type) {
		case *ast.ListValue:
			if expected, ok := ret.ExpectedTypes[node].(*schema.ListType); ok {
				for _, value := range node.Values {
					ret.ExpectedTypes[value] = expected.Type
				}
			}
		case *ast.ObjectValue:
			switch expected := ret.ExpectedTypes[node].(type) {
			case *schema.InputObjectType:
				for _, field := range node.Fields {
					if expected, ok := expected.Fields[field.Name.Name]; ok {
						ret.ExpectedTypes[field.Value] = expected.Type
						if expected.DefaultValue != nil {
							ret.DefaultValues[field.Value] = expected.DefaultValue
						}
					}
				}
			case *schema.ObjectType:
				for _, field := range node.Fields {
					if expected, ok := expected.Fields[field.Name.Name]; ok {
						ret.ExpectedTypes[field.Value] = expected.Type
					}
				}
			}
		case *ast.Directive:
			if directive := s.DirectiveDefinition(node.Name.Name); directive != nil {
				for _, arg := range node.Arguments {
					if expected, ok := directive.Arguments[arg.Name.Name]; ok {
						ret.ExpectedTypes[arg.Value] = expected.Type
						if expected.DefaultValue != nil {
							ret.DefaultValues[arg.Value] = expected.DefaultValue
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
			selectionSetScope = field.Type
		case *ast.FragmentDefinition:
			selectionSetScope = s.NamedType(node.TypeCondition.Name.Name)
		case *ast.InlineFragment:
			if node.TypeCondition == nil {
				selectionSetScope = selectionSetScopes[len(selectionSetScopes)-1]
			} else {
				selectionSetScope = s.NamedType(node.TypeCondition.Name.Name)
			}
		case *ast.OperationDefinition:
			if op := node.OperationType; op == nil || *op == ast.OperationTypeQuery {
				selectionSetScope = s.QueryType()
			} else if *op == ast.OperationTypeMutation {
				selectionSetScope = s.MutationType()
			} else if *op == ast.OperationTypeSubscription {
				selectionSetScope = s.SubscriptionType()
			}
		case *ast.SelectionSet:
			t := selectionSetScopes[len(selectionSetScopes)-1]
			ret.SelectionSetTypes[node] = t
			selectionSetScope = t
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
