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
}

func (d *InterfaceType) String() string {
	return d.Name
}

func (d *InterfaceType) IsInputType() bool {
	return false
}

func (d *InterfaceType) IsOutputType() bool {
	return true
}

func (d *InterfaceType) IsSubTypeOf(other Type) bool {
	return d.IsSameType(other)
}

func (d *InterfaceType) IsSameType(other Type) bool {
	return d == other
}

func (d *InterfaceType) NamedType() string {
	return d.Name
}

func (d *InterfaceType) shallowValidate() error {
	if len(d.Fields) == 0 {
		return fmt.Errorf("%v must have at least one field", d.Name)
	} else {
		for name := range d.Fields {
			if !isName(name) || strings.HasPrefix(name, "__") {
				return fmt.Errorf("illegal field name: %v", name)
			}
		}
	}
	return nil
}
