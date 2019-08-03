package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArguments_Names(t *testing.T) {
	assert.Empty(t, validateSource(t, `{node(id: "foo"){id}}`))
	assert.Len(t, validateSource(t, `{object(foo: "foo"){scalar}}`), 1)
	assert.Len(t, validateSource(t, `{__typename(foo: "foo")}`), 1)
}

func TestArguments_Uniqueness(t *testing.T) {
	assert.Empty(t, validateSource(t, `{node(id: "foo"){id}}`))
	assert.Len(t, validateSource(t, `{node(id: "foo", id: "bar"){id}}`), 1)
}

func TestArguments_RequiredArguments(t *testing.T) {
	assert.Empty(t, validateSource(t, `{node(id: "foo"){id}}`))
	assert.Len(t, validateSource(t, `{node{id}}`), 1)
	assert.Len(t, validateSource(t, `{node(id: null){id}}`), 1)
}
