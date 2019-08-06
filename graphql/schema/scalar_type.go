package schema

import (
	"fmt"

	"github.com/ccbrown/api-fu/graphql/ast"
)

type ScalarType struct {
	Name        string
	Description string
	Directives  []*Directive

	// Should return nil if coercion is impossible.
	LiteralCoercion func(ast.Value) interface{}

	// Should return nil if coercion is impossible.
	VariableValueCoercion func(interface{}) interface{}

	// Should return nil if coercion is impossible. In many cases, this can be the same as
	// VariableValueCoercion.
	ResultCoercion func(interface{}) interface{}
}

func (t *ScalarType) String() string {
	return t.Name
}

func (t *ScalarType) IsInputType() bool {
	return true
}

func (t *ScalarType) IsOutputType() bool {
	return true
}

func (t *ScalarType) IsSubTypeOf(other Type) bool {
	return t.IsSameType(other)
}

func (t *ScalarType) IsSameType(other Type) bool {
	return t == other
}

func (t *ScalarType) NamedType() string {
	return t.Name
}

func (t *ScalarType) CoerceVariableValue(v interface{}) (interface{}, error) {
	if coerced := t.VariableValueCoercion(v); coerced != nil {
		return coerced, nil
	}
	return nil, fmt.Errorf("invalid scalar value")
}

func (t *ScalarType) CoerceResult(result interface{}) (interface{}, error) {
	if coerced := t.ResultCoercion(result); coerced != nil {
		return coerced, nil
	}
	return nil, fmt.Errorf("invalid scalar result value")
}

func IsScalarType(t Type) bool {
	_, ok := t.(*ScalarType)
	return ok
}
