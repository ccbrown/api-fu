package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateFragmentDeclarations(t *testing.T) {
	t.Run("NameUniqueness", func(t *testing.T) {
		assert.Empty(t, validateSource(t, `{...f} fragment f on Object { scalar }`))
		assert.Len(t, validateSource(t, `{...f} fragment f on Object { scalar } fragment f on Object { scalar }`), 1)
	})

	t.Run("SpreadTypeExistence", func(t *testing.T) {
		assert.Empty(t, validateSource(t, `{...f} fragment f on Object { scalar }`))
		assert.Len(t, validateSource(t, `{...f} fragment f on ASDF { scalar }`), 1)
	})

	t.Run("OnCompositeTypes", func(t *testing.T) {
		assert.Empty(t, validateSource(t, `{...f} fragment f on Object { scalar }`))
		assert.Len(t, validateSource(t, `{...f} fragment f on String { scalar }`), 1)
	})

	t.Run("MustBeUsed", func(t *testing.T) {
		assert.Empty(t, validateSource(t, `{...f} fragment f on Object { scalar }`))
		assert.Len(t, validateSource(t, `{scalar} fragment f on Object { scalar }`), 1)
	})
}
