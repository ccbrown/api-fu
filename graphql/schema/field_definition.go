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

// FieldCost describes the cost of resolving a field, enabling rate limiting and metering.
type FieldCost struct {
	// If non-nil, this context will be passed on to sub-selections of the current field.
	Context context.Context

	// This is the cost of executing the resolver. Typically it will be 1, but if a resolver is
	// particularly expensive, it may be greater.
	Resolver int

	// This is a multiplier applied to all sub-selections of the current field. For fields that
	// return arrays, this is typically the number of expected results (e.g. the "first" or "last"
	// argument to a connection field). Defaults to 1 if not set.
	Multiplier int
}

// Returns a cost function which returns a constant resolver cost with no multiplier.
func FieldResolverCost(n int) func(FieldCostContext) FieldCost {
	return func(FieldCostContext) FieldCost {
		return FieldCost{
			Resolver: n,
		}
	}
}

// FieldCostContext contains important context passed to field cost functions.
type FieldCostContext struct {
	Context context.Context

	// The arguments that were provided.
	Arguments map[string]interface{}
}

// FieldDefinition defines an object's field.
type FieldDefinition struct {
	Description       string
	Arguments         map[string]*InputValueDefinition
	Type              Type
	Directives        []*Directive
	DeprecationReason string

	// This function can be used to define the cost of resolving the field. The total cost of an
	// operation can be calculated before the operation is executed, enabling rate limiting and
	// metering.
	Cost func(FieldCostContext) FieldCost

	Resolve func(FieldContext) (interface{}, error)
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
