package parser

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ccbrown/apifu/graphql/ast"
)

func TestParser_parseValue(t *testing.T) {
	for src, expected := range map[string]ast.Value{
		`null`: &ast.NullValue{},
		`[123 "abc"]`: &ast.ListValue{
			Values: []ast.Value{
				&ast.IntValue{
					Value: "123",
				},
				&ast.StringValue{
					Value: "abc",
				},
			},
		},
		`["""long""" "short"]`: &ast.ListValue{
			Values: []ast.Value{
				&ast.StringValue{
					Value: "long",
				},
				&ast.StringValue{
					Value: "short",
				},
			},
		},
		`{foo: "foo"}`: &ast.ObjectValue{
			Fields: []*ast.ObjectField{
				&ast.ObjectField{
					Name: &ast.Name{
						Name: "foo",
					},
					Value: &ast.StringValue{
						Value: "foo",
					},
				},
			},
		},
	} {
		p := newParser([]byte(src))
		actual := p.parseValue()
		assert.Empty(t, p.errors)
		assert.Equal(t, expected, actual)
		assert.Equal(t, 0, p.recursion)
	}
}

func TestParser_ParseDocument_KitchenSink(t *testing.T) {
	src, err := ioutil.ReadFile("testdata/kitchen-sink.graphql")
	require.NoError(t, err)
	doc, errs := ParseDocument(src)
	assert.Empty(t, errs)
	require.NotNil(t, doc)
	assert.NotEmpty(t, doc.Definitions)
}

func TestParser_ParseDocument_DeepRecursion(t *testing.T) {
	const nesting = 10000000
	src := strings.Repeat("{x", nesting) + strings.Repeat("}", nesting)
	_, errs := ParseDocument([]byte(src))
	assert.NotEmpty(t, errs)
	// most importantly, we shouldn't hang or overflow the stack
}
