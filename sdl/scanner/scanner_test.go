package scanner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ccbrown/api-fu/sdl/token"
)

func TestScanner(t *testing.T) {
	s := New([]byte(`{`+"\n"+`foo {`+"\r\n"+` bar: "baz"}`+"\r"+`}`), ScanIgnored)
	for _, expected := range []struct {
		Token   token.Token
		Literal string
		Line    int
		Column  int
	}{
		{token.PUNCTUATOR, "{", 1, 1},
		{token.LINE_TERMINATOR, "\n", 1, 2},
		{token.NAME, "foo", 2, 1},
		{token.WHITE_SPACE, " ", 2, 4},
		{token.PUNCTUATOR, "{", 2, 5},
		{token.LINE_TERMINATOR, "\r\n", 2, 6},
		{token.WHITE_SPACE, " ", 3, 1},
		{token.NAME, "bar", 3, 2},
		{token.PUNCTUATOR, ":", 3, 5},
		{token.WHITE_SPACE, " ", 3, 6},
		{token.STRING_VALUE, `"baz"`, 3, 7},
		{token.PUNCTUATOR, "}", 3, 12},
		{token.LINE_TERMINATOR, "\r", 3, 13},
		{token.PUNCTUATOR, "}", 4, 1},
	} {
		require.True(t, s.Scan())
		assert.Equal(t, expected.Token, s.Token())
		assert.Equal(t, expected.Literal, s.Literal())
		assert.Equal(t, expected.Line, s.Position().Line)
		assert.Equal(t, expected.Column, s.Position().Column)
	}
	assert.False(t, s.Scan())
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
	require.Len(t, s.Errors(), 1)
	err := s.Errors()[0]
	assert.Equal(t, 1, err.Line)
	assert.Equal(t, 2, err.Column)
}

func TestScanner_IllegalUTF8Character(t *testing.T) {
	s := New([]byte("\xc3"), 0)
	s.Scan()
	require.Len(t, s.Errors(), 1)
	assert.Equal(t, 1, s.Errors()[0].Column)
}

func TestScanner_Strings(t *testing.T) {
	for src, value := range map[string]string{
		`"simple"`:                           `simple`,
		`" white space "`:                    ` white space `,
		`"quote \""`:                         `quote "`,
		`"escaped \n\r\b\t\f"`:               "escaped \n\r\b\t\f",
		`"slashes \\ \/"`:                    `slashes \ /`,
		`"unicode \u1234\u5678\u90AB\uCDef"`: "unicode \u1234\u5678\u90AB\uCDEF",
	} {
		s := New([]byte(src), ScanIgnored)
		assert.True(t, s.Scan())
		assert.Equal(t, src, s.Literal())
		assert.Equal(t, value, s.StringValue())
		assert.False(t, s.Scan())
		assert.Empty(t, s.Errors())
	}

	for name, tc := range map[string]struct {
		Source              string
		ExpectedLiteral     string
		ExpectedErrorColumn int
	}{
		"BadEscapeSequence":        {`"\x"`, `"\x"`, 3},
		"BadUnicodeEscapeSequence": {`"\ufooo"`, `"\ufooo"`, 5},
		"Unterminated":             {`"foo` + "\n" + `"`, `"foo`, 5},
		"IllegalCharacter":         {`"👾"`, `"👾"`, 2},
	} {
		t.Run(name, func(t *testing.T) {
			s := New([]byte(tc.Source), 0)
			assert.True(t, s.Scan())
			assert.Equal(t, tc.ExpectedLiteral, s.Literal())
			require.NotEmpty(t, s.Errors())
			assert.NotEmpty(t, s.Errors()[0].Error())
			assert.Equal(t, 1, s.Errors()[0].Line)
			assert.Equal(t, tc.ExpectedErrorColumn, s.Errors()[0].Column)
		})
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

	t.Run("IllegalPosition", func(t *testing.T) {
		s := New([]byte("foo\ufeff"), ScanIgnored)
		assert.True(t, s.Scan())
		assert.False(t, s.Scan())
		require.Len(t, s.Errors(), 1)
		assert.Equal(t, 4, s.Errors()[0].Column)
	})
}

func TestScanner_SkipsIgnored(t *testing.T) {
	s := New([]byte("{\n foo {\n  bar\n } \n}"), 0)
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
		token.NAME,
		token.PUNCTUATOR,
		token.PUNCTUATOR,
	}, tokens)
	assert.Equal(t, []string{"{", "foo", "{", "bar", "}", "}"}, literals)
	assert.Empty(t, s.Errors())
}
