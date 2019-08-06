package scanner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ccbrown/api-fu/graphql/token"
)

func TestScanner(t *testing.T) {
	s := New([]byte(`{`+"\n"+`node(id: "foo") {`+"\r\n"+`...frag}`+"\r"+`}`), ScanIgnored)
	for _, expected := range []struct {
		Token   token.Token
		Literal string
		Line    int
		Column  int
	}{
		{token.PUNCTUATOR, "{", 1, 1},
		{token.LINE_TERMINATOR, "\n", 1, 2},
		{token.NAME, "node", 2, 1},
		{token.PUNCTUATOR, "(", 2, 5},
		{token.NAME, "id", 2, 6},
		{token.PUNCTUATOR, ":", 2, 8},
		{token.WHITE_SPACE, " ", 2, 9},
		{token.STRING_VALUE, `"foo"`, 2, 10},
		{token.PUNCTUATOR, ")", 2, 15},
		{token.WHITE_SPACE, " ", 2, 16},
		{token.PUNCTUATOR, "{", 2, 17},
		{token.LINE_TERMINATOR, "\r\n", 2, 18},
		{token.PUNCTUATOR, "...", 3, 1},
		{token.NAME, "frag", 3, 4},
		{token.PUNCTUATOR, "}", 3, 8},
		{token.LINE_TERMINATOR, "\r", 3, 9},
		{token.PUNCTUATOR, "}", 4, 1},
	} {
		require.True(t, s.Scan())
		assert.Equal(t, expected.Token, s.Token())
		assert.Equal(t, expected.Literal, s.Literal())
		assert.Equal(t, expected.Line, s.Line())
		assert.Equal(t, expected.Column, s.Column())
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
	s := New([]byte("\xc3\x28"), 0)
	s.Scan()
	require.Len(t, s.Errors(), 1)
	assert.Equal(t, 1, s.Errors()[0].Column)
}

func TestScanner_IncompleteEllipsis(t *testing.T) {
	s := New([]byte(".foo"), 0)
	assert.True(t, s.Scan())
	require.Len(t, s.Errors(), 1)
	assert.Equal(t, 2, s.Errors()[0].Column)
	assert.Equal(t, "foo", s.Literal())

	s = New([]byte("..foo"), 0)
	assert.True(t, s.Scan())
	require.Len(t, s.Errors(), 1)
	assert.Equal(t, 3, s.Errors()[0].Column)
	assert.Equal(t, "foo", s.Literal())
}

func TestScanner_Strings(t *testing.T) {
	for src, value := range map[string]string{
		`"simple"`:                                  `simple`,
		`" white space "`:                           ` white space `,
		`"quote \""`:                                `quote "`,
		`"escaped \n\r\b\t\f"`:                      "escaped \n\r\b\t\f",
		`"slashes \\ \/"`:                           `slashes \ /`,
		`"unicode \u1234\u5678\u90AB\uCDef"`:        "unicode \u1234\u5678\u90AB\uCDEF",
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
		assert.Equal(t, src, s.StringValue())
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

	t.Run("BadExponent", func(t *testing.T) {
		s := New([]byte(`123ex`), 0)
		assert.True(t, s.Scan())
		assert.Equal(t, "123e", s.Literal())
		require.NotEmpty(t, s.Errors())
		assert.Equal(t, 5, s.Errors()[0].Column)
	})
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
