package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVariablesInputTypes(t *testing.T) {
	assert.Empty(t, validateSource(t, `query ($id: ID!) {node(id: $id){id}}`))
	assert.Len(t, validateSource(t, `query ($id: Object!) {node(id: $id){id}}`), 1)
}
