package token

type Token int

const (
	INVALID Token = iota

	PUNCTUATOR
	NAME
	STRING_VALUE

	UNICODE_BOM
	WHITE_SPACE
	LINE_TERMINATOR
)

func (t Token) IsIgnored() bool {
	switch t {
	case UNICODE_BOM, WHITE_SPACE, LINE_TERMINATOR:
		return true
	default:
		return false
	}
}

type Position struct {
	Line   int
	Column int
}
