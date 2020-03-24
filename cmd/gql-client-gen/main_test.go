package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	schema, err := LoadSchema("testdata/github-schema.json")
	require.NoError(t, err)

	_, errs := Generate(schema, "test", []string{"testdata/github.go"}, "gql")
	assert.Empty(t, errs)
}
