package schema

import (
	"fmt"
	"strings"
)

type FieldDefinition struct {
	Description string
	Arguments   map[string]*InputValueDefinition
	Type        Type
	Directives  []*Directive
}

func (d *FieldDefinition) shallowValidate() error {
	if d.Type == nil {
		return fmt.Errorf("field is missing type")
	} else if !d.Type.IsOutputType() {
		return fmt.Errorf("%v cannot be used as a field type", d.Type)
	} else {
		for name := range d.Arguments {
			if !isName(name) || strings.HasPrefix(name, "__") {
				return fmt.Errorf("illegal field argument name: %v", name)
			}
		}
	}
	return nil
}
