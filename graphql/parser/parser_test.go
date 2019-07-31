package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ccbrown/go-api/graphql/ast"
)

func TestParser_ParseValue(t *testing.T) {
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
		p := New([]byte(src))
		actual := p.ParseValue()
		assert.Empty(t, p.Errors())
		assert.Equal(t, expected, actual)
	}
}
