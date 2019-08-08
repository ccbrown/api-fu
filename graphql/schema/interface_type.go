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
	ObjectType  func(object interface{}) *ObjectType
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

func (t *InterfaceType) NamedType() string {
	return t.Name
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
