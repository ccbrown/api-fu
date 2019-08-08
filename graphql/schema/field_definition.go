package schema

import (
	"context"
	"fmt"
	"strings"
)

type FieldContext struct {
	Context   context.Context
	Object    interface{}
	Arguments map[string]interface{}
}

type FieldDefinition struct {
	Description string
	Arguments   map[string]*InputValueDefinition
	Type        Type
	Directives  []*Directive

	Resolve func(*FieldContext) (interface{}, error)
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
