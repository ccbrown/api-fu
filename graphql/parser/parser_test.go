package parser

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ccbrown/go-api/graphql/ast"
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
