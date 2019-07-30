package token

type Token int

const (
	INVALID Token = iota

	PUNCTUATOR
	NAME
	INT_VALUE
	FLOAT_VALUE
	STRING_VALUE

	UNICODE_BOM
	WHITE_SPACE
	LINE_TERMINATOR
	COMMENT
	COMMA
)

// https://graphql.github.io/graphql-spec/June2018/#sec-Source-Text.Ignored-Tokens
func (t Token) IsIgnored() bool {
	switch t {
	case UNICODE_BOM, WHITE_SPACE, LINE_TERMINATOR, COMMENT, COMMA:
		return true
	default:
		return false
	}
}
