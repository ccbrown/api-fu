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
