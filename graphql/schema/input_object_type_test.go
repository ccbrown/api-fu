package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ccbrown/api-fu/graphql/parser"
)

func TestInputObjectType_Coercion(t *testing.T) {
	inputType := &InputObjectType{
		Fields: map[string]*InputValueDefinition{
			"a": &InputValueDefinition{
				Type: StringType,
			},
			"b": &InputValueDefinition{
				Type: NewNonNullType(IntType),
			},
		},
	}
	for name, tc := range map[string]struct {
		Literal        string
		VariableValues map[string]interface{}
		Expected       interface{}
	}{
		"Constants":            {`{ a: "abc", b: 123 }`, nil, map[string]interface{}{"a": "abc", "b": 123}},
		"NullAndConstant":      {`{ a: null, b: 123 }`, nil, map[string]interface{}{"a": nil, "b": 123}},
		"BConstant":            {`{ b: 123 }`, nil, map[string]interface{}{"b": 123}},
		"VarNullAndConstant":   {`{ a: $var, b: 123 }`, map[string]interface{}{"var": nil}, map[string]interface{}{"a": nil, "b": 123}},
		"VarAbsentAndConstant": {`{ a: $var, b: 123 }`, nil, map[string]interface{}{"b": 123}},
		"BVar":                 {`{ b: $var }`, map[string]interface{}{"var": 123}, map[string]interface{}{"b": 123}},
		"Var":                  {`$var`, map[string]interface{}{"var": map[string]interface{}{"b": 123}}, map[string]interface{}{"b": 123}},
		"String":               {`abc123`, nil, nil},
		"StringAndString":      {`{ a: "abc", b: "123" }`, nil, nil},
		"AString":              {`{ a: "abc" }`, nil, nil},
		"BVarAbsent":           {`{ b: $var }`, nil, nil},
		"StringAndNull":        {`{ a: "abc", b: null }`, nil, nil},
		"UnexpectedField":      {`{ b: 123, c: "xyz" }`, nil, nil},
	} {
		t.Run(name, func(t *testing.T) {
			value, errs := parser.ParseValue([]byte(tc.Literal))
			require.Empty(t, errs)
			coerced, err := CoerceLiteral(value, inputType, tc.VariableValues)
			if tc.Expected != nil {
				assert.NoError(t, err)
				assert.Equal(t, tc.Expected, coerced)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
