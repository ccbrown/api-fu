package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOperations_LoneAnonymousOperation(t *testing.T) {
	assert.Empty(t, validateSource(t, `{scalar}`))
	assert.Len(t, validateSource(t, `{scalar} {scalar}`), 1)
}

func TestOperations_NameUniqueness(t *testing.T) {
	assert.Empty(t, validateSource(t, `query foo {scalar}`))
	assert.Len(t, validateSource(t, `query foo {scalar} query foo {scalar}`), 1)
}
