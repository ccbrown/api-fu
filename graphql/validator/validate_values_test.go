package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ccbrown/api-fu/graphql/parser"
	"github.com/ccbrown/api-fu/graphql/schema"
)

func TestValues_InputObjectFieldNames(t *testing.T) {
	assert.Empty(t, validateSource(t, `{findDog(complex:{name:"fido"}){nickname}}`))
	assert.Len(t, validateSource(t, `{findDog(complex:{favoriteCookieFlavor:"bacon"}){nickname}}`), 1)
}

func TestValues_InputObjectFieldUniqueness(t *testing.T) {
	assert.Empty(t, validateSource(t, `{findDog(complex:{name:"fido"}){nickname}}`))
	assert.Len(t, validateSource(t, `{findDog(complex:{name:"fido" name:"fido"}){nickname}}`), 1)
}

func TestValues_InputObjectRequiredFields(t *testing.T) {
	assert.Empty(t, validateSource(t, `{object(object: {requiredString:""}){scalar}}`))
	assert.Len(t, validateSource(t, `{object(object: {}){scalar}}`), 1)
	assert.Len(t, validateSource(t, `{object(object: {requiredString:null}){scalar}}`), 1)
}

func TestValues_OfCorrectType(t *testing.T) {
	assert.Empty(t, validateSource(t, `{booleanArgField(booleanArg: true)}`))
	assert.Empty(t, validateSource(t, `{booleanArgField(booleanArg: null)}`))
	assert.Len(t, validateSource(t, `{booleanArgField(booleanArg: "foo")}`), 1)

	assert.Empty(t, validateSource(t, `{floatArgField(floatArg: 123)}`))
	assert.Len(t, validateSource(t, `{floatArgField(floatArg: "123")}`), 1)

	assert.Empty(t, validateSource(t, `{intArgField(intArg: 123)}`))
	assert.Len(t, validateSource(t, `{intArgField(intArg: "123")}`), 1)

	assert.Empty(t, validateSource(t, `{enumArgField(enumArg: FOO)}`))
	assert.Len(t, validateSource(t, `{enumArgField(enumArg: "FOO")}`), 1)
	assert.Len(t, validateSource(t, `{enumArgField(enumArg: ASDF)}`), 1)

	assert.Empty(t, validateSource(t, `{intListArgField(intListArg: [1])}`))
	assert.Empty(t, validateSource(t, `{intListArgField(intListArg: 1)}`))
	assert.Len(t, validateSource(t, `{intListArgField(intListArg: ["1"])}`), 1)
	assert.Len(t, validateSource(t, `{intListArgField(intListArg: "1")}`), 1)

	assert.Empty(t, validateSource(t, `{intListListArgField(intListListArg: 1)}`))
	assert.Empty(t, validateSource(t, `{intListListArgField(intListListArg: [[1]])}`))
	assert.Len(t, validateSource(t, `{intListListArgField(intListListArg: "1")}`), 1)
	assert.Len(t, validateSource(t, `{intListListArgField(intListListArg: [1])}`), 1)

	assert.Empty(t, validateSource(t, `query q ($s: ComplexInput = {name: "Fido"}) {findDog(complex:$s){nickname}}`))
	assert.Len(t, validateSource(t, `query q ($s: ComplexInput = {name: 123}) {findDog(complex:$s){nickname}}`), 1)
	assert.Len(t, validateSource(t, `query q ($s: ComplexInput = "foo") {findDog(complex:$s){nickname}}`), 1)
}

func TestValues_ValidateCoercion(t *testing.T) {
	inputObjectType := &schema.InputObjectType{
		Fields: map[string]*schema.InputValueDefinition{
			"a": &schema.InputValueDefinition{
				Type: schema.StringType,
			},
			"b": &schema.InputValueDefinition{
				Type: schema.NewNonNullType(schema.IntType),
			},
		},
	}
	for name, tc := range map[string]struct {
		Type    schema.Type
		Literal string
		Okay    bool
	}{
		"ObjectConstants":       {inputObjectType, `{ a: "abc", b: 123 }`, true},
		"ObjectNullAndConstant": {inputObjectType, `{ a: null, b: 123 }`, true},
		"ObjectBConstant":       {inputObjectType, `{ b: 123 }`, true},
		"ObjectVarAndConstant":  {inputObjectType, `{ a: $var, b: 123 }`, true},
		"ObjectBVar":            {inputObjectType, `{ b: $var }`, true},
		"ObjectVar":             {inputObjectType, `$var`, true},
		"ObjectString":          {inputObjectType, `"abc123"`, false},
		"ObjectStringAndString": {inputObjectType, `{ a: "abc", b: "123" }`, false},
		"ObjectAString":         {inputObjectType, `{ a: "abc" }`, false},
		"ObjectStringAndNull":   {inputObjectType, `{ a: "abc", b: null }`, false},
		"ObjectUnexpectedField": {inputObjectType, `{ b: 123, c: "xyz" }`, false},
		"IntList":               {schema.NewListType(schema.IntType), `[1, 2, 3]`, true},
		"MixedList":             {schema.NewListType(schema.IntType), `[1, "b", true]`, false},
		"Int":                   {schema.NewListType(schema.IntType), `1`, true},
		"Null":                  {schema.NewListType(schema.IntType), `null`, true},
		"NestedIntListList":     {schema.NewListType(schema.NewListType(schema.IntType)), `[[1], [2, 3]]`, true},
		"NestedIntList":         {schema.NewListType(schema.NewListType(schema.IntType)), `[1, 2, 3]`, false},
		"NestedInt":             {schema.NewListType(schema.NewListType(schema.IntType)), `1`, true},
		"NestedNull":            {schema.NewListType(schema.NewListType(schema.IntType)), `null`, true},
	} {
		t.Run(name, func(t *testing.T) {
			value, parseErrs := parser.ParseValue([]byte(tc.Literal))
			require.Empty(t, parseErrs)
			errs := validateCoercion(value, tc.Type, true)
			if tc.Okay {
				assert.Empty(t, errs)
			} else {
				assert.NotEmpty(t, errs)
			}
		})
	}
}
