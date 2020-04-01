package main

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	schema, err := LoadSchema("testdata/github-schema.json")
	require.NoError(t, err)

	_, errs := Generate(schema, "test", []string{"testdata/github.go"}, "gql")
	require.Empty(t, errs)
}

func TestRun(t *testing.T) {
	assert.Empty(t, Run(ioutil.Discard, "--pkg", "test", "-i", "testdata/github.go", "--schema", "testdata/github-schema.json"))
	assert.NotEmpty(t, Run(ioutil.Discard, "-i", "testdata/github.go", "--schema", "testdata/github-schema.json"))
	assert.NotEmpty(t, Run(ioutil.Discard, "--pkg", "test", "-i", "testdata/github.go"))
	assert.NotEmpty(t, Run(ioutil.Discard, "--pkg", "test", "-i", "testdata/github.go", "--schema", "testdata/not-the-github-schema.json"))
	assert.NotEmpty(t, Run(ioutil.Discard, "--pkg", "test", "-i", "testdata/github-schema.json", "--schema", "testdata/github-schema.json"))
}
