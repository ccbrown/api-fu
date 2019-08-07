package schema

import (
	"fmt"

	"github.com/ccbrown/api-fu/graphql/ast"
)

type ListType struct {
	Type Type
}

func NewListType(t Type) *ListType {
	return &ListType{
		Type: t,
	}
}

func (t *ListType) String() string {
	return "[" + t.Type.String() + "]"
}

func (t *ListType) IsInputType() bool {
	return t.Type.IsInputType()
}

func (t *ListType) IsOutputType() bool {
	return t.Type.IsOutputType()
}

func (t *ListType) IsSubTypeOf(other Type) bool {
	return t.IsSameType(other) || t.Type.IsSubTypeOf(other)
}

func (t *ListType) IsSameType(other Type) bool {
	if nn, ok := other.(*ListType); ok {
		return t.Type.IsSameType(nn.Type)
	}
	return false
}

func (t *ListType) Unwrap() Type {
	return t.Type
}

func (t *ListType) CoerceVariableValue(v interface{}) (interface{}, error) {
	switch v := v.(type) {
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, v := range v {
			if coerced, err := CoerceVariableValue(v, t.Type); err != nil {
				return nil, err
			} else {
				result[i] = coerced
			}
		}
		return result, nil
	}
	return nil, fmt.Errorf("invalid variable type")
}

func (t *ListType) CoerceLiteral(node ast.Value, variableValues map[string]interface{}) ([]interface{}, error) {
	return t.coerceLiteral(node, variableValues, true)
}

func (t *ListType) coerceLiteral(node ast.Value, variableValues map[string]interface{}, allowItemToListCoercion bool) ([]interface{}, error) {
	if listNode, ok := node.(*ast.ListValue); ok {
		result := make([]interface{}, len(listNode.Values))
		for i, v := range listNode.Values {
			if coerced, err := coerceLiteral(v, t.Type, variableValues, false); err != nil {
				return nil, err
			} else {
				result[i] = coerced
			}
		}
		return result, nil
	} else if allowItemToListCoercion {
		if coerced, err := CoerceLiteral(node, t.Type, variableValues); err != nil {
			return nil, err
		} else {
			return []interface{}{coerced}, nil
		}
	}
	return nil, fmt.Errorf("cannot coerce to %v", t)
}

func IsListType(t Type) bool {
	_, ok := t.(*ListType)
	return ok
}
