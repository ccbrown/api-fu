package schema

import (
	"fmt"
	"strings"

	"github.com/ccbrown/api-fu/graphql/ast"
)

type InputObjectType struct {
	Name        string
	Description string
	Directives  []*Directive
	Fields      map[string]*InputValueDefinition
}

func (t *InputObjectType) String() string {
	return t.Name
}

func (t *InputObjectType) IsInputType() bool {
	return true
}

func (t *InputObjectType) IsOutputType() bool {
	return false
}

func (t *InputObjectType) IsSubTypeOf(other Type) bool {
	return t.IsSameType(other)
}

func (t *InputObjectType) IsSameType(other Type) bool {
	return t == other
}

func (t *InputObjectType) TypeName() string {
	return t.Name
}

func (t *InputObjectType) CoerceVariableValue(v interface{}) (interface{}, error) {
	switch v := v.(type) {
	case map[string]interface{}:
		result := map[string]interface{}{}
		for name, field := range t.Fields {
			if fieldValue, ok := v[name]; ok {
				if coerced, err := CoerceVariableValue(fieldValue, field.Type); err != nil {
					return nil, err
				} else {
					result[name] = coerced
				}
			} else if field.DefaultValue != nil {
				if field.DefaultValue == Null {
					result[name] = nil
				} else {
					result[name] = field.DefaultValue
				}
			} else if IsNonNullType(field.Type) {
				return nil, fmt.Errorf("the %v field is required", name)
			}
		}
		for name := range v {
			if _, ok := t.Fields[name]; !ok {
				return nil, fmt.Errorf("unknown field: %v", name)
			}
		}
		return result, nil
	}
	return nil, fmt.Errorf("invalid variable type")
}

func (t *InputObjectType) CoerceLiteral(node *ast.ObjectValue, variableValues map[string]interface{}) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	for _, field := range node.Fields {
		name := field.Name.Name
		if fieldDef, ok := t.Fields[name]; !ok {
			return nil, fmt.Errorf("unknown field: %v", name)
		} else {
			if variable, ok := field.Value.(*ast.Variable); ok {
				if _, ok := variableValues[variable.Name.Name]; !ok {
					continue
				}
			}
			if coerced, err := CoerceLiteral(field.Value, fieldDef.Type, variableValues); err != nil {
				return nil, err
			} else {
				result[name] = coerced
			}
		}
	}
	for name, field := range t.Fields {
		if v, ok := result[name]; !ok && field.DefaultValue != nil {
			if field.DefaultValue == Null {
				result[name] = nil
			} else {
				result[name] = field.DefaultValue
			}
		} else if (!ok || v == nil) && IsNonNullType(field.Type) {
			return nil, fmt.Errorf("the %v field is required", name)
		}
	}
	return result, nil
}

func (t *InputObjectType) shallowValidate() error {
	if len(t.Fields) == 0 {
		return fmt.Errorf("%v must have at least one field", t.Name)
	} else {
		for name, field := range t.Fields {
			if !isName(name) || strings.HasPrefix(name, "__") {
				return fmt.Errorf("illegal field name: %v", name)
			} else if !field.Type.IsInputType() {
				return fmt.Errorf("%v field must be an input type", name)
			}
		}
	}
	return nil
}
