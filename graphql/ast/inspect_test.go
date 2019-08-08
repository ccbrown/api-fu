package ast_test

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ccbrown/api-fu/graphql/ast"
	"github.com/ccbrown/api-fu/graphql/parser"
)

func TestInspect_KitchenSink(t *testing.T) {
	src, err := ioutil.ReadFile("testdata/kitchen-sink.graphql")
	require.NoError(t, err)
	doc, errs := parser.ParseDocument(src)
	require.Empty(t, errs)
	ast.Inspect(doc, func(node ast.Node) bool {
		return true
	})
}
