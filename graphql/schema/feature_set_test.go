package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFeatureSet(t *testing.T) {
	s := NewFeatureSet("a", "b", "c")
	assert.True(t, s.Has("a"))
	assert.True(t, s.Has("b"))
	assert.True(t, s.Has("c"))
	assert.False(t, s.Has("d"))

	s2 := NewFeatureSet("a", "b")
	assert.True(t, s2.IsSubsetOf(s))
	assert.False(t, s.IsSubsetOf(s2))
}

func TestFeatureSet_Nil(t *testing.T) {
	var s FeatureSet
	assert.False(t, s.Has("a"))

	s2 := NewFeatureSet("a", "b")
	assert.True(t, s.IsSubsetOf(s2))
	assert.False(t, s2.IsSubsetOf(s))

	var s3 FeatureSet
	assert.True(t, s.IsSubsetOf(s3))
	assert.True(t, s3.IsSubsetOf(s))
}
