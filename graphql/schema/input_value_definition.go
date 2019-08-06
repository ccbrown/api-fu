package schema

import "fmt"

type InputValueDefinition struct {
	Description string
	Type        Type

	// For null, set this to Null.
	DefaultValue interface{}

	Directives []*Directive
}

type explicitNull struct{}

// Used to specify an explicit "null" default for input values.
var Null = (*explicitNull)(nil)

func (d *InputValueDefinition) shallowValidate() error {
	if d.Type == nil {
		return fmt.Errorf("input value is missing type")
	} else if !d.Type.IsInputType() {
		return fmt.Errorf("%v cannot be used as an input value type", d.Type)
	}
	return nil
}
