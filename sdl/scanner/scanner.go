package scanner

import (
	"fmt"
	"unicode/utf8"

	"github.com/ccbrown/api-fu/sdl/token"
)

type Error struct {
	Message string
	Line    int
	Column  int
}

func (err *Error) Error() string {
	return err.Message
}

type Scanner struct {
	src    []byte
	mode   Mode
	offset int
	line   int
	column int
	errors []*Error

	nextRune     rune
	nextRuneSize int

	token            token.Token
	tokenOffset      int
	tokenPosition    token.Position
	tokenLength      int
	tokenStringValue string
}

type Mode uint

const (
	ScanIgnored Mode = 1 << iota
)

func New(src []byte, mode Mode) *Scanner {
	s := &Scanner{
		src:    src,
		mode:   mode,
		line:   1,
		column: 1,
	}
	s.readNextRune()
	return s
}

func (s *Scanner) Errors() []*Error {
	return s.errors
}

func (s *Scanner) errorf(message string, args ...interface{}) {
	s.errors = append(s.errors, &Error{
		Message: fmt.Sprintf(message, args...),
		Line:    s.line,
		Column:  s.column,
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
	if r == '\n' || (r == '\r' && s.nextRune != '\n') {
		s.line++
		s.column = 1
	} else {
		s.column++
	}
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

func (s *Scanner) isDone() bool {
	return len(s.src) == s.offset
}

func (s *Scanner) Scan() bool {
	for {
		s.token = token.INVALID
		s.tokenOffset = s.offset
		s.tokenPosition = token.Position{
			Line:   s.line,
			Column: s.column,
		}

		if s.isDone() {
			return false
		}

		switch s.nextRune {
		case '\t', ' ':
			s.consumeRune()
			s.token = token.WHITE_SPACE
		case ':', '=', '{', '}', '!':
			s.consumeRune()
			s.token = token.PUNCTUATOR
		case '\r', '\n':
			if s.consumeRune() == '\r' && s.nextRune == '\n' {
				s.consumeRune()
			}
			s.token = token.LINE_TERMINATOR
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
			if s.consumeName() {
				s.token = token.NAME
			} else {
				s.errorf("illegal character: %#U", s.nextRune)
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

func (s *Scanner) Position() token.Position {
	return s.tokenPosition
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
