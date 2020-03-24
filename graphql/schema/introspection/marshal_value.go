package introspection

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/ccbrown/api-fu/graphql/schema"
)

// Used to marshal input types for default value introspection.
func marshalValue(t schema.Type, v interface{}) (string, error) {
	if v == schema.Null {
		return "null", nil
	}

	switch t := t.(type) {
	case *schema.ScalarType:
		b, err := json.Marshal(v)
		return string(b), err
	case *schema.ListType:
		v := reflect.ValueOf(v)
		if v.Kind() != reflect.Slice {
			return "", fmt.Errorf("default value is not a slice")
		}
		parts := make([]string, v.Len())
		for i := range parts {
			s, err := marshalValue(t.Type, v.Index(i).Interface())
			if err != nil {
				return "", err
			}
			parts[i] = s
		}
		return "[" + strings.Join(parts, ", ") + "]", nil
	case *schema.InputObjectType:
		if t.ResultCoercion == nil {
			return "", fmt.Errorf("%v cannot be serialized", t.Name)
		}
		kv, err := t.ResultCoercion(v)
		if err != nil {
			return "", err
		}
		parts := make([]string, 0, len(kv))
		for k, v := range kv {
			s, err := marshalValue(t.Fields[k].Type, v)
			if err != nil {
				return "", err
			}
			parts = append(parts, k+": "+s)
		}
		return "{" + strings.Join(parts, ", ") + "}", nil
	case *schema.EnumType:
		return t.CoerceResult(v)
	case *schema.NonNullType:
		return marshalValue(t.Type, v)
	default:
		return "", fmt.Errorf("unsupported value type: %T", t)
	}
}
