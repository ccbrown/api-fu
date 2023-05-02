package schema

import (
	"fmt"
	"reflect"
)

// Inspect traverses the types referenced by the schema, invoking f for each one. If f returns true,
// Inspect will recursively inspect the types referenced by the given node. For many schemas,
// this means f must be able to break cycles to prevent Inspect from running infinitely.
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
		for _, node := range n.MemberTypes {
			Inspect(node, f)
		}
	case *InterfaceType:
		for _, node := range n.Fields {
			Inspect(node, f)
		}
	case *InputObjectType:
		for _, node := range n.Fields {
			Inspect(node, f)
		}
	case *ObjectType:
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
	case *InputValueDefinition:
		Inspect(n.Type, f)
	case *Directive:
		Inspect(n.Definition, f)
	case *DirectiveDefinition:
		for _, node := range n.Arguments {
			Inspect(node, f)
		}
	case *ListType:
		Inspect(n.Type, f)
	case *NonNullType:
		Inspect(n.Type, f)
	case *EnumType:
		for _, node := range n.Directives {
			Inspect(node, f)
		}
	case *ScalarType:
		for _, node := range n.Directives {
			Inspect(node, f)
		}
	default:
		panic(fmt.Errorf("unknown node type: %T", n))
	}

	f(nil)
}
