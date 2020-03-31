package main

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	schema, err := LoadSchema("testdata/github-schema.json")
	require.NoError(t, err)

	_, errs := Generate(schema, "test", []string{"testdata/github.go"}, "gql")
	require.Empty(t, errs)

	Run(ioutil.Discard, "--pkg", "test", "-i", "testdata/github.go", "--schema", "testdata/github-schema.json")
}
