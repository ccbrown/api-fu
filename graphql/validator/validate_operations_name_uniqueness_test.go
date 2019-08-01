package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOperationsNameUniqueness(t *testing.T) {
	assert.Empty(t, validateSource(t, `query foo {scalar}`))
	assert.NotEmpty(t, validateSource(t, `query foo {scalar} query foo {scalar}`))
}
