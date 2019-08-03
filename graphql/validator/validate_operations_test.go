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

func TestOperations_SubscriptionSingleRootField(t *testing.T) {
	assert.Empty(t, validateSource(t, `subscription sub {object{int int2}}`))
	assert.Len(t, validateSource(t, `subscription sub {int int2}`), 1)

	assert.Empty(t, validateSource(t, `subscription sub {...f} fragment f on Object {object{int int2}}`))
	assert.Len(t, validateSource(t, `subscription sub {...f} fragment f on Object {int int2}`), 1)

	assert.Len(t, validateSource(t, `subscription sub {int __typename}`), 1)
}
