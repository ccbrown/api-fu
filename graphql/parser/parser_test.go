package parser

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ccbrown/api-fu/graphql/ast"
	"github.com/ccbrown/api-fu/graphql/token"
)

func TestParseValue(t *testing.T) {
	for src, expected := range map[string]ast.Value{
		`null`: &ast.NullValue{
			Literal: token.Position{1, 1},
		},
		`[123 "abc"]`: &ast.ListValue{
			Values: []ast.Value{
				&ast.IntValue{
					Value:   "123",
					Literal: token.Position{1, 2},
				},
				&ast.StringValue{
					Value:   "abc",
					Literal: token.Position{1, 6},
				},
			},
			Opening: token.Position{1, 1},
			Closing: token.Position{1, 11},
		},
		`["""long""" "short"]`: &ast.ListValue{
			Values: []ast.Value{
				&ast.StringValue{
					Value:   "long",
					Literal: token.Position{1, 2},
				},
				&ast.StringValue{
					Value:   "short",
					Literal: token.Position{1, 13},
				},
			},
			Opening: token.Position{1, 1},
			Closing: token.Position{1, 20},
		},
		`{foo: "foo"}`: &ast.ObjectValue{
			Fields: []*ast.ObjectField{
				&ast.ObjectField{
					Name: &ast.Name{
						Name:         "foo",
						NamePosition: token.Position{1, 2},
					},
					Value: &ast.StringValue{
						Value:   "foo",
						Literal: token.Position{1, 7},
					},
				},
			},
			Opening: token.Position{1, 1},
			Closing: token.Position{1, 12},
		},
	} {
		actual, errs := ParseValue([]byte(src))
		assert.Empty(t, errs)
		assert.Equal(t, expected, actual)
	}

	t.Run("Error", func(t *testing.T) {
		_, errs := ParseValue([]byte(`!`))
		assert.Len(t, errs, 1)
	})
}

func TestParseDocument_KitchenSink(t *testing.T) {
	src, err := ioutil.ReadFile("testdata/kitchen-sink.graphql")
	require.NoError(t, err)
	doc, errs := ParseDocument(src)
	assert.Empty(t, errs)
	require.NotNil(t, doc)
	assert.NotEmpty(t, doc.Definitions)
	ast.Inspect(doc, func(node ast.Node) bool {
		switch node.(type) {
		case nil, *ast.Document:
		default:
			assert.NotEqual(t, 0, node.Position().Line)
			assert.NotEqual(t, 0, node.Position().Column)
		}
		return true
	})
}

func TestParseDocument_DeepRecursion(t *testing.T) {
	const nesting = 100000000
	src := strings.Repeat("{x", nesting) + strings.Repeat("}", nesting)
	_, errs := ParseDocument([]byte(src))
	assert.NotEmpty(t, errs)
	// most importantly, we shouldn't hang or overflow the stack
}

func TestParseDocument_ConstantValues(t *testing.T) {
	_, errs := ParseDocument([]byte(`query ($n:Int=1) {x}`))
	assert.Empty(t, errs)

	_, errs = ParseDocument([]byte(`query ($n:Float=1.2) {x}`))
	assert.Empty(t, errs)

	_, errs = ParseDocument([]byte(`query ($n:Int=$x) {x}`))
	assert.NotEmpty(t, errs)
}

func TestParseDocument_Types(t *testing.T) {
	doc, errs := ParseDocument([]byte(`query ($n:Float, $ns:[[Int!]]) {x}`))
	assert.Empty(t, errs)

	require.Len(t, doc.Definitions, 1)

	op := doc.Definitions[0].(*ast.OperationDefinition)
	require.Len(t, op.VariableDefinitions, 2)

	_, ok := op.VariableDefinitions[0].Type.(*ast.NamedType)
	assert.True(t, ok)

	list, ok := op.VariableDefinitions[1].Type.(*ast.ListType)
	require.True(t, ok)
	list, ok = list.Type.(*ast.ListType)
	require.True(t, ok)
	nonNull, ok := list.Type.(*ast.NonNullType)
	require.True(t, ok)
	named, ok := nonNull.Type.(*ast.NamedType)
	assert.True(t, ok)
	assert.Equal(t, "Int", named.Name.Name)
}

func TestParseDocument_Errors(t *testing.T) {
	for name, tc := range map[string]struct {
		Source         string
		ExpectedLine   int
		ExpectedColumn int
	}{
		"EOF":                             {`{`, 1, 2},
		"EmptyDocument":                   {``, 1, 1},
		"FragmentNamedOn":                 {`fragment on on Foo {x}`, 1, 10},
		"BadOperationType":                {`foo foo {x}`, 1, 1},
		"ExpectedSelectionSet":            {`{...}`, 1, 5},
		"ExpectedSelection":               {`{}`, 1, 2},
		"ExpectedTypeCondition":           {`fragment foo {x}`, 1, 14},
		"ExpectedArgument":                {`{foo()}`, 1, 6},
		"ExpectedVariableDefinition":      {`query q() {x}`, 1, 9},
		"ExpectedVariableDefinitionColon": {`query q($foo) {x}`, 1, 13},
		"ExpectedListTypeTerminator":      {`query q($foo: [Int x]) {x}`, 1, 20},
		"ExpectedArgumentColon":           {`{x(foo)}`, 1, 7},
		"ExpectedObjectColon":             {`{x(o: {foo})}`, 1, 11},
		"ExpectedVariable":                {`query q(x)`, 1, 9},
		"ScannerError":                    {`{👾x}`, 1, 2},
	} {
		t.Run(name, func(t *testing.T) {
			_, errs := ParseDocument([]byte(tc.Source))
			require.Len(t, errs, 1)
			assert.NotEmpty(t, errs[0].Error())
			assert.Equal(t, tc.ExpectedLine, errs[0].Line)
			assert.Equal(t, tc.ExpectedColumn, errs[0].Column)
		})
	}
}
