package schema

import "fmt"

type EnumType struct {
	Name        string
	Description string
	Directives  []*Directive
	Values      map[string]*EnumValueDefinition
}

type EnumValueDefinition struct {
	Description string
	Directives  []*Directive
}

func (t *EnumType) String() string {
	return t.Name
}

func (t *EnumType) IsInputType() bool {
	return true
}

func (t *EnumType) IsOutputType() bool {
	return true
}

func (t *EnumType) IsSubTypeOf(other Type) bool {
	return t.IsSameType(other)
}

func (t *EnumType) IsSameType(other Type) bool {
	return t == other
}

func (t *EnumType) NamedType() string {
	return t.Name
}

func (d *EnumType) shallowValidate() error {
	if len(d.Values) == 0 {
		return fmt.Errorf("%v must have at least one field", d.Name)
	} else {
		for name := range d.Values {
			if !isName(name) || name == "true" || name == "false" || name == "null" {
				return fmt.Errorf("illegal field name: %v", name)
			}
		}
	}
	return nil
}

func IsEnumType(t Type) bool {
	_, ok := t.(*EnumType)
	return ok
}
