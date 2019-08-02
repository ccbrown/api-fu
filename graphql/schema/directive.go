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
	Name        string
	Description string
	Arguments   map[string]InputValueDefinition
	Locations   []DirectiveLocation
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
	if name := d.Name; !isName(name) || strings.HasPrefix(name, "__") {
		return fmt.Errorf("illegal directive name: %v", name)
	}
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
