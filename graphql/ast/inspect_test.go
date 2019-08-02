package ast_test

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ccbrown/apifu/graphql/ast"
	"github.com/ccbrown/apifu/graphql/parser"
)

func TestInspect_KitchenSink(t *testing.T) {
	src, err := ioutil.ReadFile("testdata/kitchen-sink.graphql")
	require.NoError(t, err)
	doc, errs := parser.ParseDocument(src)
	require.Empty(t, errs)
	ast.Inspect(doc, func(node interface{}) bool {
		return true
	})
}
