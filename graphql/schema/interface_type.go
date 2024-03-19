package schema

import (
	"fmt"
	"strings"
)

type InterfaceType struct {
	Name        string
	Description string
	Directives  []*Directive
	Fields      map[string]*FieldDefinition

	// This type is only available for introspection and use when the given features are enabled.
	RequiredFeatures FeatureSet
}

func (t *InterfaceType) GetField(name string, features FeatureSet) *FieldDefinition {
	if field, ok := t.Fields[name]; ok && field.RequiredFeatures.IsSubsetOf(features) {
		return field
	}
	return nil
}

func (t *InterfaceType) String() string {
	return t.Name
}

func (t *InterfaceType) IsInputType() bool {
	return false
}

func (t *InterfaceType) IsOutputType() bool {
	return true
}

func (t *InterfaceType) IsSubTypeOf(other Type) bool {
	return t.IsSameType(other)
}

func (t *InterfaceType) IsSameType(other Type) bool {
	return t == other
}

func (t *InterfaceType) TypeRequiredFeatures() FeatureSet {
	return t.RequiredFeatures
}

func (t *InterfaceType) TypeName() string {
	return t.Name
}

func (t *InterfaceType) shallowValidate() error {
	hasAtLeastOneUnconditionalField := false
	for name, field := range t.Fields {
		if !isName(name) || strings.HasPrefix(name, "__") {
			return fmt.Errorf("illegal field name: %v", name)
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
	return nil
}
