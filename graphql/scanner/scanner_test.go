package scanner

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ccbrown/go-api/graphql/token"
)

func TestScanner(t *testing.T) {
	s := New([]byte(`{ node(id: "foo") {}}`), ScanIgnored)
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
	assert.Equal(t, []string{"{", " ", "node", "(", "id", ":", " ", `"foo"`, ")", " ", "{", "}", "}"}, literals)
	assert.Empty(t, s.Errors())
}

func TestScanner_IllegalCharacter(t *testing.T) {
	s := New([]byte(`{😃}`), 0)
	var tokens []token.Token
	var literals []string
	for s.Scan() {
		tokens = append(tokens, s.Token())
		literals = append(literals, s.Literal())
	}
	assert.Equal(t, []token.Token{token.PUNCTUATOR, token.PUNCTUATOR}, tokens)
	assert.Equal(t, []string{"{", "}"}, literals)
	assert.Len(t, s.Errors(), 1)
}

func TestScanner_Strings(t *testing.T) {
	for src, value := range map[string]string{
		`"simple"`:                                  `simple`,
		`" white space "`:                           ` white space `,
		`"quote \""`:                                `quote "`,
		`"escaped \n\r\b\t\f"`:                      "escaped \n\r\b\t\f",
		`"slashes \\ \/"`:                           `slashes \ /`,
		`"unicode \u1234\u5678\u90AB\uCDEF"`:        "unicode \u1234\u5678\u90AB\uCDEF",
		`"""simple"""`:                              `simple`,
		`""" white space """`:                       ` white space `,
		`"""contains " quote"""`:                    `contains " quote`,
		`"""contains \""" triplequote"""`:           `contains """ triplequote`,
		`"""multi` + "\n" + `line"""`:               "multi\nline",
		`"""` + "multi\rline\r\nnormalized" + `"""`: "multi\nline\nnormalized",
		`"""unescaped \n\r\b\t\f\u1234"""`:          `unescaped \n\r\b\t\f\u1234`,
		`"""slashes \\ \/"""`:                       `slashes \\ \/`,
		`"""

          spans
            multiple
              lines

          """`: "spans\n  multiple\n    lines",
		`"""trailing triplequote \""""""`: `trailing triplequote """`,
	} {
		s := New([]byte(src), ScanIgnored)
		assert.True(t, s.Scan())
		assert.Equal(t, src, s.Literal())
		assert.Equal(t, value, s.StringValue())
		assert.False(t, s.Scan())
		assert.Empty(t, s.Errors())
	}
}

func TestScanner_Ints(t *testing.T) {
	for _, src := range []string{
		"4",
		"-4",
		"9",
		"0",
	} {
		s := New([]byte(src), ScanIgnored)
		assert.True(t, s.Scan())
		assert.Equal(t, src, s.Literal())
		assert.False(t, s.Scan())
		assert.Empty(t, s.Errors())
	}
}

func TestScanner_Floats(t *testing.T) {
	for _, src := range []string{
		"4.123",
		"-4.123",
		"0.123",
		"123e4",
		"123E4",
		"123e-4",
		"123e+4",
		"-123E4",
		"-123e-4",
		"-123e+4",
		"-123e4567",
	} {
		s := New([]byte(src), ScanIgnored)
		assert.True(t, s.Scan())
		assert.Equal(t, src, s.Literal())
		assert.False(t, s.Scan())
		assert.Empty(t, s.Errors())
	}
}

func TestScanner_BOM(t *testing.T) {
	s := New([]byte("\ufefffoo"), ScanIgnored)
	var tokens []token.Token
	for s.Scan() {
		tokens = append(tokens, s.Token())
	}
	assert.Equal(t, []token.Token{token.UNICODE_BOM, token.NAME}, tokens)
	assert.Empty(t, s.Errors())
}

func TestScanner_SkipsIgnored(t *testing.T) {
	s := New([]byte("{\n node {\n  #foo\n },\n}"), 0)
	var tokens []token.Token
	var literals []string
	for s.Scan() {
		tokens = append(tokens, s.Token())
		literals = append(literals, s.Literal())
	}
	assert.Equal(t, []token.Token{
		token.PUNCTUATOR,
		token.NAME,
		token.PUNCTUATOR,
		token.PUNCTUATOR,
		token.PUNCTUATOR,
	}, tokens)
	assert.Equal(t, []string{"{", "node", "{", "}", "}"}, literals)
	assert.Empty(t, s.Errors())
}
