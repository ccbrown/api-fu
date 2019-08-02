package schema

import (
	"fmt"
	"strings"
)

type InputObjectType struct {
	Name        string
	Description string
	Directives  []*Directive
	Fields      map[string]*InputValueDefinition
}

func (d *InputObjectType) String() string {
	return d.Name
}

func (d *InputObjectType) IsInputType() bool {
	return true
}

func (d *InputObjectType) IsOutputType() bool {
	return false
}

func (d *InputObjectType) IsSubTypeOf(other Type) bool {
	return d.IsSameType(other)
}

func (d *InputObjectType) IsSameType(other Type) bool {
	return d == other
}

func (d *InputObjectType) NamedType() string {
	return d.Name
}

func (d *InputObjectType) shallowValidate() error {
	if len(d.Fields) == 0 {
		return fmt.Errorf("%v must have at least one field", d.Name)
	} else {
		for name, field := range d.Fields {
			if !isName(name) || strings.HasPrefix(name, "__") {
				return fmt.Errorf("illegal field name: %v", name)
			} else if !field.Type.IsInputType() {
				return fmt.Errorf("%v field must be an input type", name)
			}
		}
	}
	return nil
}
