package ast

import (
	"fmt"
	"reflect"
)

func Inspect(node interface{}, f func(interface{}) bool) {
	if node == nil || reflect.ValueOf(node).IsNil() || !f(node) {
		return
	}

	switch n := node.(type) {
	case *Document:
		for _, node := range n.Definitions {
			Inspect(node, f)
		}
	case *OperationDefinition:
		Inspect(n.Name, f)
		for _, node := range n.VariableDefinitions {
			Inspect(node, f)
		}
		for _, node := range n.Directives {
			Inspect(node, f)
		}
		Inspect(n.SelectionSet, f)
	case *FragmentDefinition:
		Inspect(n.Name, f)
		for _, node := range n.Directives {
			Inspect(node, f)
		}
		Inspect(n.SelectionSet, f)
	case *VariableDefinition:
		Inspect(n.Variable, f)
		Inspect(n.Type, f)
		Inspect(n.DefaultValue, f)
	case *ListType:
		Inspect(n.Type, f)
	case *NonNullType:
		Inspect(n.Type, f)
	case *Directive:
		Inspect(n.Name, f)
		for _, node := range n.Arguments {
			Inspect(node, f)
		}
	case *SelectionSet:
		for _, node := range n.Selections {
			Inspect(node, f)
		}
	case *Field:
		Inspect(n.Alias, f)
		Inspect(n.Name, f)
		for _, node := range n.Arguments {
			Inspect(node, f)
		}
		for _, node := range n.Directives {
			Inspect(node, f)
		}
		Inspect(n.SelectionSet, f)
	case *FragmentSpread:
		Inspect(n.FragmentName, f)
		for _, node := range n.Directives {
			Inspect(node, f)
		}
	case *InlineFragment:
		Inspect(n.TypeCondition, f)
		for _, node := range n.Directives {
			Inspect(node, f)
		}
		Inspect(n.SelectionSet, f)
	case *Argument:
		Inspect(n.Name, f)
		Inspect(n.Value, f)
	case *NamedType:
		Inspect(n.Name, f)
	case *Variable:
		Inspect(n.Name, f)
	case *Name, *BooleanValue, *IntValue, *FloatValue, *StringValue, *EnumValue, *NullValue:
	case *ListValue:
		for _, node := range n.Values {
			Inspect(node, f)
		}
	case *ObjectValue:
		for _, node := range n.Fields {
			Inspect(node, f)
		}
	case *ObjectField:
		Inspect(n.Name, f)
		Inspect(n.Value, f)
	default:
		panic(fmt.Errorf("unknown node type: %T", n))
	}

	f(nil)
}
