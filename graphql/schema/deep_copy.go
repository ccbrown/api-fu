package schema

import "fmt"

func deepCopySchemaDefinition(def *SchemaDefinition) *SchemaDefinition {
	newNamedTypes := make(map[string]NamedType)

	// Create shallow copies for all the named types.
	Inspect(def, func(node any) bool {
		if node, ok := node.(NamedType); ok {
			if _, ok := newNamedTypes[node.TypeName()]; ok {
				return false
			}
			switch t := node.(type) {
			case *UnionType:
				copy := *t
				newNamedTypes[t.Name] = &copy
			case *InterfaceType:
				copy := *t
				newNamedTypes[t.Name] = &copy
			case *InputObjectType:
				copy := *t
				newNamedTypes[t.Name] = &copy
			case *ObjectType:
				copy := *t
				newNamedTypes[t.Name] = &copy
			case *EnumType:
				copy := *t
				newNamedTypes[t.Name] = &copy
			case *ScalarType:
				copy := *t
				newNamedTypes[t.Name] = &copy
			default:
				panic(fmt.Errorf("unknown named type type: %T", t))
			}
		}

		return true
	})

	// Now update all of those shallow copies to point to each other.
	for _, t := range newNamedTypes {
		fixNamedTypePointers(t, newNamedTypes)
	}

	ret := &SchemaDefinition{}
	if def.Query != nil {
		ret.Query = newNamedTypes[def.Query.Name].(*ObjectType)
	}
	if def.Mutation != nil {
		ret.Mutation = newNamedTypes[def.Mutation.Name].(*ObjectType)
	}
	if def.Subscription != nil {
		ret.Subscription = newNamedTypes[def.Subscription.Name].(*ObjectType)
	}

	if def.Directives != nil {
		ret.Directives = make(map[string]*DirectiveDefinition, len(def.Directives))
		for k, v := range def.Directives {
			newValue := *v
			fixNamedTypePointers(&newValue, newNamedTypes)
			ret.Directives[k] = &newValue
		}
	}

	if def.AdditionalTypes != nil {
		ret.AdditionalTypes = make([]NamedType, len(def.AdditionalTypes))
		for i, v := range def.AdditionalTypes {
			ret.AdditionalTypes[i] = newNamedTypes[v.TypeName()]
		}
	}

	return ret
}

func fixTypePointer(t Type, namedTypes map[string]NamedType) Type {
	switch t := t.(type) {
	case NamedType:
		if _, ok := BuiltInTypes[t.TypeName()]; ok {
			return t
		} else if ret, ok := namedTypes[t.TypeName()]; ok {
			return ret
		}
		return t
	case *NonNullType:
		return NewNonNullType(fixTypePointer(t.Unwrap(), namedTypes))
	case *ListType:
		return NewListType(fixTypePointer(t.Unwrap(), namedTypes))
	default:
		panic(fmt.Errorf("unknown named type type: %T", t))
	}
	return nil
}

// Updates pointers to named types to those contained in the given map. This function does not
// recurse into descendant named types.
func fixNamedTypePointers(node any, namedTypes map[string]NamedType) {
	switch n := node.(type) {
	case *UnionType:
		if n.Directives != nil {
			newValues := make([]*Directive, len(n.Directives))
			for i, v := range n.Directives {
				newValue := *v
				fixNamedTypePointers(&newValue, namedTypes)
				newValues[i] = &newValue
			}
			n.Directives = newValues
		}
		if n.MemberTypes != nil {
			newValues := make([]*ObjectType, len(n.MemberTypes))
			for i, v := range n.MemberTypes {
				if newValue, ok := namedTypes[v.Name].(*ObjectType); ok {
					newValues[i] = newValue
				} else {
					newValues[i] = v
				}
			}
			n.MemberTypes = newValues
		}
	case *InterfaceType:
		if n.Directives != nil {
			newValues := make([]*Directive, len(n.Directives))
			for i, v := range n.Directives {
				newValue := *v
				fixNamedTypePointers(&newValue, namedTypes)
				newValues[i] = &newValue
			}
			n.Directives = newValues
		}
		if n.Fields != nil {
			newValues := make(map[string]*FieldDefinition, len(n.Fields))
			for k, v := range n.Fields {
				newField := *v
				fixNamedTypePointers(&newField, namedTypes)
				newValues[k] = &newField
			}
			n.Fields = newValues
		}
	case *InputObjectType:
		if n.Directives != nil {
			newValues := make([]*Directive, len(n.Directives))
			for i, v := range n.Directives {
				newValue := *v
				fixNamedTypePointers(&newValue, namedTypes)
				newValues[i] = &newValue
			}
			n.Directives = newValues
		}
		if n.Fields != nil {
			newValues := make(map[string]*InputValueDefinition, len(n.Fields))
			for k, v := range n.Fields {
				newField := *v
				fixNamedTypePointers(&newField, namedTypes)
				newValues[k] = &newField
			}
			n.Fields = newValues
		}
	case *ObjectType:
		if n.Directives != nil {
			newValues := make([]*Directive, len(n.Directives))
			for i, v := range n.Directives {
				newValue := *v
				fixNamedTypePointers(&newValue, namedTypes)
				newValues[i] = &newValue
			}
			n.Directives = newValues
		}
		if n.Fields != nil {
			newValues := make(map[string]*FieldDefinition, len(n.Fields))
			for k, v := range n.Fields {
				newField := *v
				fixNamedTypePointers(&newField, namedTypes)
				newValues[k] = &newField
			}
			n.Fields = newValues
		}
		if n.ImplementedInterfaces != nil {
			newValues := make([]*InterfaceType, len(n.ImplementedInterfaces))
			for i, v := range n.ImplementedInterfaces {
				if newValue, ok := namedTypes[v.Name].(*InterfaceType); ok {
					newValues[i] = newValue
				} else {
					newValues[i] = v
				}
			}
			n.ImplementedInterfaces = newValues
		}
	case *FieldDefinition:
		if n.Directives != nil {
			newValues := make([]*Directive, len(n.Directives))
			for i, v := range n.Directives {
				newValue := *v
				fixNamedTypePointers(&newValue, namedTypes)
				newValues[i] = &newValue
			}
			n.Directives = newValues
		}
		n.Type = fixTypePointer(n.Type, namedTypes)
		if n.Arguments != nil {
			newValues := make(map[string]*InputValueDefinition, len(n.Arguments))
			for k, v := range n.Arguments {
				newField := *v
				fixNamedTypePointers(&newField, namedTypes)
				newValues[k] = &newField
			}
			n.Arguments = newValues
		}
	case *InputValueDefinition:
		if n.Directives != nil {
			newValues := make([]*Directive, len(n.Directives))
			for i, v := range n.Directives {
				newValue := *v
				fixNamedTypePointers(&newValue, namedTypes)
				newValues[i] = &newValue
			}
			n.Directives = newValues
		}
		n.Type = fixTypePointer(n.Type, namedTypes)
	case *Directive:
		if n.Definition != nil {
			newDefinition := *n.Definition
			fixNamedTypePointers(&newDefinition, namedTypes)
			n.Definition = &newDefinition
		}
	case *DirectiveDefinition:
		if n.Arguments != nil {
			newValues := make(map[string]*InputValueDefinition, len(n.Arguments))
			for k, v := range n.Arguments {
				newField := *v
				fixNamedTypePointers(&newField, namedTypes)
				newValues[k] = &newField
			}
			n.Arguments = newValues
		}
	case *EnumType:
		if n.Directives != nil {
			newValues := make([]*Directive, len(n.Directives))
			for i, v := range n.Directives {
				newValue := *v
				fixNamedTypePointers(&newValue, namedTypes)
				newValues[i] = &newValue
			}
			n.Directives = newValues
		}
		if n.Values != nil {
			newValues := make(map[string]*EnumValueDefinition, len(n.Values))
			for k, v := range n.Values {
				newValue := *v
				newValues[k] = &newValue
			}
			n.Values = newValues
		}
	case *ScalarType:
		if n.Directives != nil {
			newValues := make([]*Directive, len(n.Directives))
			for i, v := range n.Directives {
				newValue := *v
				fixNamedTypePointers(&newValue, namedTypes)
				newValues[i] = &newValue
			}
			n.Directives = newValues
		}
	default:
		panic(fmt.Errorf("unexpected node type: %T", n))
	}
}
