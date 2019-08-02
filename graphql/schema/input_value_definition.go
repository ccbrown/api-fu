package schema

import "fmt"

type InputValueDefinition struct {
	Description  string
	Type         Type
	DefaultValue interface{}
	Directives   []*Directive
}

func (d *InputValueDefinition) shallowValidate() error {
	if d.Type == nil {
		return fmt.Errorf("input value is missing type")
	} else if !d.Type.IsInputType() {
		return fmt.Errorf("%v cannot be used as an input value type", d.Type)
	}
	return nil
}
