package scanner

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ccbrown/go-api/graphql/token"
)

func TestScanner(t *testing.T) {
	s := New([]byte(`{ node(id: "foo") {}}`))
	var tokens []token.Token
	var literals []string
	for s.Scan() {
		tokens = append(tokens, s.Token())
		literals = append(literals, s.Literal())
	}
	assert.Equal(t, []token.Token{
		token.PUNCTUATOR,
		token.WHITE_SPACE,
		token.NAME,
		token.PUNCTUATOR,
		token.NAME,
		token.PUNCTUATOR,
		token.WHITE_SPACE,
		token.STRING_VALUE,
		token.PUNCTUATOR,
		token.WHITE_SPACE,
		token.PUNCTUATOR,
		token.PUNCTUATOR,
		token.PUNCTUATOR,
	}, tokens)
	assert.Equal(t, []string{
		"{",
		" ",
		"node",
		"(",
		"id",
		":",
		" ",
		`"foo"`,
		")",
		" ",
		"{",
		"}",
		"}",
	}, literals)
	assert.Nil(t, s.Error())
}
