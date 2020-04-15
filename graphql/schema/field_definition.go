package schema

import (
	"context"
	"fmt"
	"strings"
)

// FieldContext contains important context passed to resolver implementations.
type FieldContext struct {
	Context   context.Context
	Schema    *Schema
	Object    interface{}
	Arguments map[string]interface{}

	// IsSubscribe is true if this is a subscription field being invoked for a subscribe operation.
	// Subselections of this field will not be executed, and the return value will be returned
	// immediately to the caller of Subscribe.
	IsSubscribe bool
}

// FieldDefinition defines an object's field.
type FieldDefinition struct {
	Description       string
	Arguments         map[string]*InputValueDefinition
	Type              Type
	Directives        []*Directive
	DeprecationReason string

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
