package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
