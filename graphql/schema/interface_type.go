package schema

import (
	"context"
	"fmt"
	"strings"
)

type InterfaceType struct {
	Name        string
	Description string
	Directives  []*Directive
	Fields      map[string]*FieldDefinition

	// If given, this type will only be visible via introspection if the given function returns
	// true. This can for example be used to build APIs that are gated behind feature flags.
	IsVisible func(context.Context) bool
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

func (t *InterfaceType) TypeName() string {
	return t.Name
}

func (t *InterfaceType) IsTypeVisible(ctx context.Context) bool {
	if t.IsVisible == nil {
		return true
	}
	return t.IsVisible(ctx)
}

func (t *InterfaceType) shallowValidate() error {
	if len(t.Fields) == 0 {
		return fmt.Errorf("%v must have at least one field", t.Name)
	} else {
		for name := range t.Fields {
			if !isName(name) || strings.HasPrefix(name, "__") {
				return fmt.Errorf("illegal field name: %v", name)
			}
		}
	}
	return nil
}
