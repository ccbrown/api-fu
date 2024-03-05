package schema

import (
	"context"
	"fmt"
	"strings"

	"github.com/ccbrown/api-fu/graphql/ast"
)

type InputObjectType struct {
	Name        string
	Description string
	Directives  []*Directive
	Fields      map[string]*InputValueDefinition

	// If given, input objects can validated and converted to other types via this function.
	// Otherwise the objects will remain as maps. This function is called after all fields are fully
	// coerced.
	InputCoercion func(map[string]interface{}) (interface{}, error)

	// Normally input objects only need to be coerced from inputs. However, if an argument of this
	// type is given a default value, we need to be able to do the reverse in order to serialize it
	// for introspection queries.
	//
	// For most use-cases, this function is optional. If it is required, but nil, you will get an
	// error when you attempt to create the schema.
	ResultCoercion func(interface{}) (map[string]interface{}, error)

	// If given, this type will only be visible via introspection if the given function returns
	// true. This can for example be used to build APIs that are gated behind feature flags.
	IsVisible func(context.Context) bool
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

func (t *InputObjectType) IsTypeVisible(ctx context.Context) bool {
	if t.IsVisible == nil {
		return true
	}
	return t.IsVisible(ctx)
}

func (t *InputObjectType) CoerceVariableValue(v interface{}) (interface{}, error) {
	result := map[string]interface{}{}

	switch v := v.(type) {
	case map[string]interface{}:
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
	default:
		return nil, fmt.Errorf("invalid variable type")
	}

	if t.InputCoercion != nil {
		return t.InputCoercion(result)
	}
	return result, nil
}

func (t *InputObjectType) CoerceLiteral(node *ast.ObjectValue, variableValues map[string]interface{}) (interface{}, error) {
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

	if t.InputCoercion != nil {
		return t.InputCoercion(result)
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
