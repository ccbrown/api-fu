package schema

import (
	"fmt"

	"github.com/ccbrown/api-fu/graphql/ast"
)

type EnumType struct {
	Name        string
	Description string
	Directives  []*Directive
	Values      map[string]*EnumValueDefinition
}

type EnumValueDefinition struct {
	Description string
	Directives  []*Directive
	Value       interface{}
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

func (t *EnumType) shallowValidate() error {
	if len(t.Values) == 0 {
		return fmt.Errorf("%v must have at least one field", t.Name)
	} else {
		for name := range t.Values {
			if !isName(name) || name == "true" || name == "false" || name == "null" {
				return fmt.Errorf("illegal field name: %v", name)
			}
		}
	}
	return nil
}

func (t *EnumType) CoerceVariableValue(v interface{}) (interface{}, error) {
	if s, ok := v.(string); ok {
		if def, ok := t.Values[s]; ok {
			return def.Value, nil
		}
	}
	return nil, fmt.Errorf("invalid enum value")
}

func (t *EnumType) CoerceLiteral(from ast.Value) (interface{}, error) {
	if from, ok := from.(*ast.EnumValue); ok {
		if v, ok := t.Values[from.Value]; ok {
			return v.Value, nil
		}
	}
	return nil, fmt.Errorf("invalid enum value")
}

func (t *EnumType) CoerceResult(result interface{}) (string, error) {
	for name, def := range t.Values {
		if def.Value == result {
			return name, nil
		}
	}
	return "", fmt.Errorf("invalid enum result value")
}

func IsEnumType(t Type) bool {
	_, ok := t.(*EnumType)
	return ok
}
