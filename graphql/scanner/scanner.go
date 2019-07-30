package scanner

import (
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
	offset int
	errors []*Error

	nextRune     rune
	nextRuneSize int

	token       token.Token
	tokenOffset int
	tokenLength int
}

func New(src []byte) *Scanner {
	s := &Scanner{
		src: src,
	}
	s.readNextRune()
	return s
}

func (s *Scanner) Errors() []*Error {
	return s.errors
}

func (s *Scanner) error(message string) {
	s.errors = append(s.errors, &Error{
		message: message,
	})
}

func (s *Scanner) readNextRune() {
	if s.isDone() {
		s.nextRune = -1
		s.nextRuneSize = 0
	} else if r, size := utf8.DecodeRune(s.src[s.offset:]); r == utf8.RuneError && size != 0 {
		s.error("invalid utf8 sequence")
		s.nextRune = r
		s.nextRuneSize = 1
	} else {
		s.nextRune = r
		s.nextRuneSize = size
	}
}

func (s *Scanner) peek() rune {
	r, _ := utf8.DecodeRune(s.src[s.offset:])
	return r
}

func (s *Scanner) consumeRune() {
	s.offset += s.nextRuneSize
	s.readNextRune()
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

func (s *Scanner) consumeString() bool {
	if s.nextRune != '"' {
		return false
	}
	s.consumeRune()
	terminated := false
	for !s.isDone() {
		if s.nextRune == '"' {
			s.consumeRune()
			terminated = true
			break
		}
		s.consumeRune()
	}
	if !terminated {
		s.error("unterminated string")
	}
	return true
}

const maxErrors = 10

func (s *Scanner) isDone() bool {
	return len(s.errors) >= maxErrors || len(s.src) == s.offset
}

func (s *Scanner) Scan() bool {
	s.token = token.INVALID

	for {
		if s.isDone() {
			return false
		}

		s.tokenOffset = s.offset

		switch s.nextRune {
		case '\t', ' ':
			s.consumeRune()
			s.token = token.WHITE_SPACE
		case '!', '$', '(', ')', ':', '=', '@', '[', ']', '{', '|', '}':
			s.consumeRune()
			s.token = token.PUNCTUATOR
		case '.':
			s.consumeRune()
			if s.nextRune == '.' && s.peek() == '.' {
				s.consumeRune()
				s.consumeRune()
				s.token = token.PUNCTUATOR
			} else {
				s.error("illegal character")
			}
		case '"':
			s.consumeString()
			s.token = token.STRING_VALUE
		default:
			if s.consumeName() {
				s.token = token.NAME
			} else {
				s.consumeRune()
				s.error("illegal character")
			}
		}

		if s.token != token.INVALID {
			s.tokenLength = s.offset - s.tokenOffset
			return true
		}
	}
}

func (s *Scanner) Token() token.Token {
	return s.token
}

func (s *Scanner) Literal() string {
	return string(s.src[s.tokenOffset : s.tokenOffset+s.tokenLength])
}
