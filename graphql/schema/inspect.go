package schema

import (
	"fmt"
	"reflect"
)

func Inspect(node interface{}, f func(interface{}) bool) {
	if node == nil || reflect.ValueOf(node).IsNil() || !f(node) {
		return
	}

	switch n := node.(type) {
	case *SchemaDefinition:
		for _, node := range n.Directives {
			Inspect(node, f)
		}
		Inspect(n.Query, f)
		Inspect(n.Mutation, f)
		Inspect(n.Subscription, f)
		for _, node := range n.AdditionalTypes {
			Inspect(node, f)
		}
	case *UnionType:
		for _, node := range n.Directives {
			Inspect(node, f)
		}
		for _, node := range n.MemberTypes {
			Inspect(node, f)
		}
	case *InterfaceType:
		for _, node := range n.Directives {
			Inspect(node, f)
		}
		for _, node := range n.Fields {
			Inspect(node, f)
		}
	case *InputObjectType:
		for _, node := range n.Directives {
			Inspect(node, f)
		}
		for _, node := range n.Fields {
			Inspect(node, f)
		}
	case *ObjectType:
		for _, node := range n.Directives {
			Inspect(node, f)
		}
		for _, node := range n.Fields {
			Inspect(node, f)
		}
		for _, node := range n.ImplementedInterfaces {
			Inspect(node, f)
		}
	case *FieldDefinition:
		Inspect(n.Type, f)
		for _, node := range n.Arguments {
			Inspect(node, f)
		}
		for _, node := range n.Directives {
			Inspect(node, f)
		}
	case *InputValueDefinition:
		Inspect(n.Type, f)
		for _, node := range n.Directives {
			Inspect(node, f)
		}
	case *DirectiveDefinition:
		for _, node := range n.Arguments {
			Inspect(node, f)
		}
	case *ListType:
		Inspect(n.Type, f)
	case *NonNullType:
		Inspect(n.Type, f)
	case *EnumType, *ScalarType:
	default:
		panic(fmt.Errorf("unknown node type: %T", n))
	}

	f(nil)
}
