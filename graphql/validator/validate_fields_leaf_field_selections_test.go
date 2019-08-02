package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFieldsLeafFieldSelections(t *testing.T) {
	assert.Empty(t, validateSource(t, `{scalar}`))
	assert.Len(t, validateSource(t, `{scalar{asd}}`), 1)

	assert.Empty(t, validateSource(t, `{interface{scalar}}`))
	assert.Len(t, validateSource(t, `{interface}`), 1)
}
