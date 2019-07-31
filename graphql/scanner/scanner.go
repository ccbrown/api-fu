package scanner

import (
	"fmt"
	"unicode/utf8"

	"github.com/ccbrown/go-api/graphql/token"
)

type Error struct {
	message string
}

func (err *Error) Error() string {
	return err.message
}

type Scanner struct {
	src    []byte
	mode   Mode
	offset int
	errors []*Error

	nextRune     rune
	nextRuneSize int

	token            token.Token
	tokenOffset      int
	tokenLength      int
	tokenStringValue string
}

type Mode uint

const (
	ScanIgnored Mode = 1 << iota
)

func New(src []byte, mode Mode) *Scanner {
	s := &Scanner{
		src:  src,
		mode: mode,
	}
	s.readNextRune()
	return s
}

func (s *Scanner) Errors() []*Error {
	return s.errors
}

func (s *Scanner) errorf(message string, args ...interface{}) {
	s.errors = append(s.errors, &Error{
		message: fmt.Sprintf(message, args...),
	})
}

func (s *Scanner) readNextRune() {
	if s.isDone() {
		s.nextRune = -1
		s.nextRuneSize = 0
	} else if r, size := utf8.DecodeRune(s.src[s.offset:]); r == utf8.RuneError && size != 0 {
		s.nextRune = r
		s.nextRuneSize = 1
	} else {
		s.nextRune = r
		s.nextRuneSize = size
	}
}

func (s *Scanner) peek() rune {
	r, _ := utf8.DecodeRune(s.src[s.offset+s.nextRuneSize:])
	return r
}

func (s *Scanner) consumeRune() rune {
	r := s.nextRune
	s.offset += s.nextRuneSize
	s.readNextRune()
	return r
}

func (s *Scanner) consumeName() bool {
	if r := s.nextRune; r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
		s.consumeRune()
		for !s.isDone() {
			if r := s.nextRune; r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
				s.consumeRune()
			} else {
				break
			}
		}
		return true
	}
	return false
}

func isSourceCharacter(r rune) bool {
	return r == '\t' || r == '\n' || r == '\r' || (r >= 0x20 && r <= 0xffff)
}

const maxErrors = 10

func (s *Scanner) isDone() bool {
	return len(s.errors) >= maxErrors || len(s.src) == s.offset
}

func (s *Scanner) Scan() bool {
	for {
		if s.isDone() {
			return false
		}

		s.token = token.INVALID
		s.tokenOffset = s.offset

		switch s.nextRune {
		case '\t', ' ':
			s.consumeRune()
			s.token = token.WHITE_SPACE
		case '!', '$', '(', ')', ':', '=', '@', '[', ']', '{', '|', '}':
			s.consumeRune()
			s.token = token.PUNCTUATOR
		case ',':
			s.consumeRune()
			s.token = token.COMMA
		case '\r', '\n':
			if s.consumeRune() == '\r' && s.nextRune == '\n' {
				s.consumeRune()
			}
			s.token = token.LINE_TERMINATOR
		case '#':
			for s.nextRune != '\r' && s.nextRune != '\n' {
				s.consumeRune()
			}
			s.token = token.COMMENT
		case '.':
			s.consumeRune()
			if s.nextRune == '.' && s.peek() == '.' {
				s.consumeRune()
				s.consumeRune()
				s.token = token.PUNCTUATOR
			} else {
				s.errorf("illegal character")
			}
		case '"':
			s.tokenStringValue = s.consumeStringValue()
			s.token = token.STRING_VALUE
		case utf8.RuneError:
			s.errorf("invalid utf-8 character")
			s.consumeRune()
		case 0xfeff:
			if s.offset == 0 {
				s.token = token.UNICODE_BOM
			} else {
				s.errorf("illegal byte order mark")
			}
			s.consumeRune()
		default:
			if s.consumeIntegerPart() {
				if s.consumeFractionalPart() {
					s.consumeExponentPart()
					s.token = token.FLOAT_VALUE
				} else if s.consumeExponentPart() {
					s.token = token.FLOAT_VALUE
				} else {
					s.token = token.INT_VALUE
				}
			} else if s.consumeName() {
				s.token = token.NAME
			} else {
				s.errorf("illegal character %#U", s.nextRune)
				s.consumeRune()
			}
		}

		if s.token == token.INVALID || (s.token.IsIgnored() && (s.mode&ScanIgnored) == 0) {
			continue
		}

		s.tokenLength = s.offset - s.tokenOffset
		return true
	}
}

func (s *Scanner) Token() token.Token {
	return s.token
}

func (s *Scanner) Literal() string {
	return string(s.src[s.tokenOffset : s.tokenOffset+s.tokenLength])
}

func (s *Scanner) StringValue() string {
	if s.token == token.STRING_VALUE {
		return s.tokenStringValue
	} else {
		return s.Literal()
	}
}
