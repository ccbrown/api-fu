package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDirectives_Defined(t *testing.T) {
	assert.Empty(t, validateSource(t, `{scalar @include(if: true)}`))
	assert.Len(t, validateSource(t, `{scalar @foo(if: true)}`), 1)
}

func TestDirectives_InValidLocations(t *testing.T) {
	assert.Empty(t, validateSource(t, `{scalar @include(if: true)}`))
	assert.Len(t, validateSource(t, `query @include(if: true) {scalar}`), 1)
}

func TestDirectives_UniquePerLocation(t *testing.T) {
	assert.Empty(t, validateSource(t, `{scalar @include(if: true)}`))
	assert.Len(t, validateSource(t, `{scalar @include(if: true) @include(if: true)}`), 1)
}
