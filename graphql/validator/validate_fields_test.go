package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFields_LeafFieldSelections(t *testing.T) {
	assert.Empty(t, validateSource(t, `{scalar}`))
	assert.Len(t, validateSource(t, `{scalar{asd}}`), 1)

	assert.Empty(t, validateSource(t, `{interface{scalar}}`))
	assert.Len(t, validateSource(t, `{interface}`), 1)
}

func TestFields_SelectionsOnObjectsInterfacesAndUnions(t *testing.T) {
	assert.Empty(t, validateSource(t, `{object{scalar}}`))
	assert.Len(t, validateSource(t, `{object{asd}}`), 1)

	assert.Empty(t, validateSource(t, `{interface{scalar}}`))
	assert.Len(t, validateSource(t, `{interface{asd}}`), 1)

	assert.Empty(t, validateSource(t, `{union{__typename}}`))
	assert.Len(t, validateSource(t, `{union{a}}`), 1)
}
