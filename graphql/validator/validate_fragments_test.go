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

		assert.Empty(t, validateSource(t, `{... on Object {scalar}}`))
		assert.Len(t, validateSource(t, `{... on ASDF {scalar}}`), 1)
	})

	t.Run("OnCompositeTypes", func(t *testing.T) {
		assert.Empty(t, validateSource(t, `{...f} fragment f on Object { scalar }`))
		assert.Len(t, validateSource(t, `{...f} fragment f on String { scalar }`), 1)

		assert.Empty(t, validateSource(t, `{... on Object { scalar }}`))
		assert.Len(t, validateSource(t, `{... on String { scalar }}`), 1)
	})

	t.Run("MustBeUsed", func(t *testing.T) {
		assert.Empty(t, validateSource(t, `{...f} fragment f on Object { scalar }`))
		assert.Len(t, validateSource(t, `{scalar} fragment f on Object { scalar }`), 1)
	})
}

func TestValidateFragmentSpreads(t *testing.T) {
	t.Run("TargetDefined", func(t *testing.T) {
		assert.Empty(t, validateSource(t, `{...f} fragment f on Object { scalar }`))
		assert.Len(t, validateSource(t, `{...asdf}`), 1)
	})

	t.Run("MustNotFormCycles", func(t *testing.T) {
		assert.Empty(t, validateSource(t, `{...a} fragment a on Object { scalar }`))
		assert.Len(t, validateSource(t, `{...a} fragment a on Object {...a}`), 1)
		assert.Len(t, validateSource(t, `{...a} fragment a on Object {...b} fragment b on Object {...a}`), 2)
	})

	t.Run("IsPossible", func(t *testing.T) {
		assert.Empty(t, validateSource(t, `{...{int}}`))
		assert.Len(t, validateSource(t, `{...{asdf}}`), 1)

		assert.Empty(t, validateSource(t, `{...f} fragment f on Object {... on Object {scalar}}`))
		assert.Len(t, validateSource(t, `{...f} fragment f on Object {... on Object2 {scalar}}`), 1)
		assert.Empty(t, validateSource(t, `{objects{...f}} fragment f on Object {... on Object {scalar}}`))

		assert.Empty(t, validateSource(t, `{resource{...obj}} fragment abstract on Node {id} fragment obj on Resource {...abstract}`))
		assert.Empty(t, validateSource(t, `{union{...obj}} fragment abstract on Union {... on UnionObjectB {b}} fragment obj on UnionObjectA {...abstract}`))

		assert.Empty(t, validateSource(t, `{union{...abstract}} fragment abstract on Union {... on UnionObjectB {b}}`))
		assert.Empty(t, validateSource(t, `{resource{...abstract}} fragment abstract on Node {... on Resource {id}}`))

		assert.Len(t, validateSource(t, `{union{...abstract}} fragment abstract on Union {... on Object {scalar}}`), 1)
		assert.Len(t, validateSource(t, `{resource{...abstract}} fragment abstract on Node {... on Object {scalar}}`), 1)

		assert.Empty(t, validateSource(t, `{union{... on UnionMember{... on Union{... on UnionObjectA{a}}}}}`))
		assert.Len(t, validateSource(t, `{union{... on UnionMember{... on Node{id}}}}`), 1)
	})

	t.Run("Features", func(t *testing.T) {
		assert.Empty(t, validateSource(t, `{experimentalObject{...a}} fragment a on ExperimentalObject { foo }`, "experimentalobject"))
		assert.Len(t, validateSource(t, `{experimentalObject{...a}} fragment a on ExperimentalObject { foo }`), 2)
	})
}
