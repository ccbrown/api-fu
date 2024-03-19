package schema

import (
	"fmt"
	"strings"
)

type ObjectType struct {
	Name        string
	Description string
	Directives  []*Directive
	Fields      map[string]*FieldDefinition

	// This type is only available for introspection and use when the given features are enabled.
	RequiredFeatures FeatureSet

	ImplementedInterfaces []*InterfaceType

	// Objects that implement one or more interfaces must define this. The function should return
	// true if obj is an object of this type.
	IsTypeOf func(obj interface{}) bool
}

func (t *ObjectType) GetField(name string, features FeatureSet) *FieldDefinition {
	if field, ok := t.Fields[name]; ok && field.RequiredFeatures.IsSubsetOf(features) {
		return field
	}
	return nil
}

func (t *ObjectType) String() string {
	return t.Name
}

func (t *ObjectType) IsInputType() bool {
	return false
}

func (t *ObjectType) IsOutputType() bool {
	return true
}

func (t *ObjectType) IsSubTypeOf(other Type) bool {
	if t.IsSameType(other) {
		return true
	} else if union, ok := other.(*UnionType); ok {
		for _, member := range union.MemberTypes {
			if t.IsSameType(member) {
				return true
			}
		}
	} else {
		for _, iface := range t.ImplementedInterfaces {
			if iface.IsSameType(other) {
				return true
			}
		}
	}
	return false
}

func (t *ObjectType) IsSameType(other Type) bool {
	return t == other
}

func (t *ObjectType) TypeRequiredFeatures() FeatureSet {
	return t.RequiredFeatures
}

func (t *ObjectType) TypeName() string {
	return t.Name
}

func (t *ObjectType) satisfyInterface(iface *InterfaceType) error {
	for name, ifaceField := range iface.Fields {
		field, ok := t.Fields[name]
		if !ok {
			return fmt.Errorf("object is missing field named %v", name)
		} else if !field.Type.IsSubTypeOf(ifaceField.Type) {
			return fmt.Errorf("object's %v field is not a subtype of the corresponding interface field", name)
		} else if !field.RequiredFeatures.IsSubsetOf(ifaceField.RequiredFeatures) {
			return fmt.Errorf("object's %v field requires features that are not required by the corresponding interface field", name)
		}
		for argName, ifaceArg := range ifaceField.Arguments {
			arg, ok := field.Arguments[argName]
			if !ok {
				return fmt.Errorf("object's %v field is missing argument named %v", name, argName)
			} else if !arg.Type.IsSameType(ifaceArg.Type) {
				return fmt.Errorf("object's %v field %v argument is not the same type as the corresponding interface argument", name, argName)
			}
		}
		for argName, arg := range field.Arguments {
			if _, ok := ifaceField.Arguments[argName]; !ok && IsNonNullType(arg.Type) {
				return fmt.Errorf("object's %v field %v argument cannot be non-null", name, argName)
			}
		}
	}
	return nil
}

func (t *ObjectType) shallowValidate() error {
	hasAtLeastOneUnconditionalField := false
	for name, field := range t.Fields {
		if !isName(name) || strings.HasPrefix(name, "__") {
			return fmt.Errorf("illegal field name: %v", name)
		} else if !field.Type.IsOutputType() {
			return fmt.Errorf("%v field must be an output type", name)
		}
		if field.RequiredFeatures.IsSubsetOf(t.RequiredFeatures) {
			hasAtLeastOneUnconditionalField = true
		}

		fieldRequiredFeatures := field.RequiredFeatures.Union(t.RequiredFeatures)
		if !field.Type.TypeRequiredFeatures().IsSubsetOf(fieldRequiredFeatures) {
			return fmt.Errorf("field type requires features that are not required by the field")
		} else {
			for name, arg := range field.Arguments {
				if !arg.Type.TypeRequiredFeatures().IsSubsetOf(fieldRequiredFeatures) {
					return fmt.Errorf("field argument %v requires features that are not required by the field", name)
				}
			}
		}
	}
	if !hasAtLeastOneUnconditionalField {
		return fmt.Errorf("%v must have at least one field", t.Name)
	}
	for _, iface := range t.ImplementedInterfaces {
		if err := t.satisfyInterface(iface); err != nil {
			return fmt.Errorf("%v does not satisfy %v: %v", t.Name, iface.Name, err.Error())
		}
	}
	if len(t.ImplementedInterfaces) > 0 && t.IsTypeOf == nil {
		return fmt.Errorf("%v implements an interface, but does not define IsTypeOf", t.Name)
	}
	return nil
}

func IsObjectType(t Type) bool {
	_, ok := t.(*ObjectType)
	return ok
}
