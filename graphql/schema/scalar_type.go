package schema

import "github.com/ccbrown/api-fu/graphql/ast"

type ScalarType struct {
	Name        string
	Description string
	Directives  []*Directive

	// Should return nil if coercion is impossible.
	CoerceLiteral func(ast.Value) interface{}
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

func IsScalarType(t Type) bool {
	_, ok := t.(*ScalarType)
	return ok
}
