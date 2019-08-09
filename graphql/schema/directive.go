package schema

import (
	"fmt"
	"strings"
)

type DirectiveLocation string

const (
	DirectiveLocationQuery              DirectiveLocation = "QUERY"
	DirectiveLocationMutation           DirectiveLocation = "MUTATION"
	DirectiveLocationSubscription       DirectiveLocation = "SUBSCRIPTION"
	DirectiveLocationField              DirectiveLocation = "FIELD"
	DirectiveLocationFragmentDefinition DirectiveLocation = "FRAGMENT_DEFINITION"
	DirectiveLocationFragmentSpread     DirectiveLocation = "FRAGMENT_SPREAD"
	DirectiveLocationInlineFragment     DirectiveLocation = "INLINE_FRAGMENT"

	DirectiveLocationSchema               DirectiveLocation = "SCHEMA"
	DirectiveLocationScalar               DirectiveLocation = "SCALAR"
	DirectiveLocationObject               DirectiveLocation = "OBJECT"
	DirectiveLocationFieldDefinition      DirectiveLocation = "FIELD_DEFINITION"
	DirectiveLocationArgumentDefinition   DirectiveLocation = "ARGUMENT_DEFINITION"
	DirectiveLocationInterface            DirectiveLocation = "INTERFACE"
	DirectiveLocationUnion                DirectiveLocation = "UNION"
	DirectiveLocationEnum                 DirectiveLocation = "ENUM"
	DirectiveLocationEnumValue            DirectiveLocation = "ENUM_VALUE"
	DirectiveLocationInputObject          DirectiveLocation = "INPUT_OBJECT"
	DirectiveLocationInputFieldDefinition DirectiveLocation = "INPUT_FIELD_DEFINITION"
)

type DirectiveDefinition struct {
	Description string
	Arguments   map[string]*InputValueDefinition
	Locations   []DirectiveLocation

	// If non-nil, this function will be invoked during field collection for each selection with
	// this directive present. If the function returns false, the selection will be skipped.
	FieldCollectionFilter func(arguments map[string]interface{}) bool
}

func referencesDirective(node interface{}, directive *DirectiveDefinition) bool {
	visited := map[interface{}]struct{}{}
	foundReference := false

	Inspect(node, func(node interface{}) bool {
		if _, ok := visited[node]; ok {
			return false
		}
		visited[node] = struct{}{}
		if node == directive {
			foundReference = true
		}
		return !foundReference
	})

	return foundReference
}

func (d *DirectiveDefinition) shallowValidate() error {
	for name, arg := range d.Arguments {
		if !isName(name) || strings.HasPrefix(name, "__") {
			return fmt.Errorf("illegal directive argument name: %v", name)
		} else if referencesDirective(arg, d) {
			return fmt.Errorf("directive is self-referencing via %v argument", name)
		}
	}
	if len(d.Locations) == 0 {
		return fmt.Errorf("directives must have one or more locations")
	}
	return nil
}

type Directive struct {
	Definition *DirectiveDefinition
	Arguments  []*Argument
}
