package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVariablesNameUniqueness(t *testing.T) {
	assert.Empty(t, validateSource(t, `query ($id: ID!) {node(id: $id){id}}`))
	assert.NotEmpty(t, validateSource(t, `query ($id: ID!, $id: ID!) {node(id: $id){id}}`))
}
