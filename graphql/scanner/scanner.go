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

func newError(message string) *Error {
	return &Error{
		message: message,
	}
}

type Scanner struct {
	src    []byte
	offset int
	err    *Error

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

func (s *Scanner) Error() *Error {
	return s.err
}

func (s *Scanner) readNextRune() {
	if r, size := utf8.DecodeRune(s.src[s.offset:]); r == utf8.RuneError && size != 0 {
		s.nextRune = r
		s.nextRuneSize = 0
		s.err = newError("invalid utf8 sequence")
	} else {
		s.nextRune = r
		s.nextRuneSize = size
	}
}

func (s *Scanner) clone() *Scanner {
	ret := *s
	return &ret
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
	if !terminated && s.err == nil {
		s.err = newError("unterminated string")
	}
	return true
}

func (s *Scanner) isDone() bool {
	return s.err != nil || len(s.src) == s.offset
}

func (s *Scanner) Scan() bool {
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
		s.token = token.ILLEGAL
		if s.nextRune == '.' {
			temp := s.clone()
			temp.consumeRune()
			if temp.nextRune == '.' {
				s.consumeRune()
				s.consumeRune()
				s.token = token.PUNCTUATOR
			}
		}
	case '"':
		s.consumeString()
		s.token = token.STRING_VALUE
	default:
		if s.consumeName() {
			s.token = token.NAME
		} else {
			s.consumeRune()
			s.token = token.ILLEGAL
		}
	}

	s.tokenLength = s.offset - s.tokenOffset
	return s.err == nil
}

func (s *Scanner) Token() token.Token {
	return s.token
}

func (s *Scanner) Literal() string {
	return string(s.src[s.tokenOffset : s.tokenOffset+s.tokenLength])
}
