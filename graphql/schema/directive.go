package schema

import (
	"fmt"
	"strings"
)

type DirectiveLocation string

const (
	DirectiveLocationQuery              = "QUERY"
	DirectiveLocationMutation           = "MUTATION"
	DirectiveLocationSubscription       = "SUBSCRIPTION"
	DirectiveLocationField              = "FIELD"
	DirectiveLocationFragmentDefinition = "FRAGMENT_DEFINITION"
	DirectiveLocationFragmentSpread     = "FRAGMENT_SPREAD"
	DirectiveLocationInlineFragment     = "INLINE_FRAGMENT"
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
	return nil
}

type Directive struct {
	Definition *DirectiveDefinition
	Arguments  []*Argument
}

var SkipDirective = &DirectiveDefinition{
	Description: "The @skip directive may be provided for fields, fragment spreads, and inline fragments, and allows for conditional exclusion during execution as described by the if argument.",
	Arguments: map[string]*InputValueDefinition{
		"if": &InputValueDefinition{
			Type: NewNonNullType(BooleanType),
		},
	},
	Locations: []DirectiveLocation{DirectiveLocationField, DirectiveLocationFragmentSpread, DirectiveLocationInlineFragment},
	FieldCollectionFilter: func(arguments map[string]interface{}) bool {
		return !arguments["if"].(bool)
	},
}

var IncludeDirective = &DirectiveDefinition{
	Description: "The @include directive may be provided for fields, fragment spreads, and inline fragments, and allows for conditional inclusion during execution as described by the if argument.",
	Arguments: map[string]*InputValueDefinition{
		"if": &InputValueDefinition{
			Type: NewNonNullType(BooleanType),
		},
	},
	Locations: []DirectiveLocation{DirectiveLocationField, DirectiveLocationFragmentSpread, DirectiveLocationInlineFragment},
	FieldCollectionFilter: func(arguments map[string]interface{}) bool {
		return arguments["if"].(bool)
	},
}
