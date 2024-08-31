package scanner

func hexRuneValue(r rune) rune {
	if r >= '0' && r <= '9' {
		return r - '0'
	} else if r >= 'a' && r <= 'f' {
		return 10 + r - 'a'
	} else if r >= 'A' && r <= 'F' {
		return 10 + r - 'A'
	}
	return -1
}

func (s *Scanner) consumeStringValue() string {
	s.consumeRune() // '"'

	value := ""

	terminated := false
	isEscaped := false
	for !terminated && !s.isDone() {
		if isEscaped {
			consumed := false
			switch s.nextRune {
			case '"', '\\', '/':
				value += string(s.nextRune)
			case 'b':
				value += string('\b')
			case 'f':
				value += string('\f')
			case 'n':
				value += string('\n')
			case 'r':
				value += string('\r')
			case 't':
				value += string('\t')
			case 'u':
				s.consumeRune()
				consumed = true

				var code rune
				for i := 0; i < 4; i++ {
					if v := hexRuneValue(s.nextRune); v < 0 {
						s.errorf("illegal unicode escape sequence")
						break
					} else {
						code = (code << 4) | v
						s.consumeRune()
					}
				}
				value += string(code)
			default:
				s.errorf("illegal escape sequence")
			}
			if !consumed {
				s.consumeRune()
			}
			isEscaped = false
			continue
		}

		if s.nextRune == '\n' || s.nextRune == '\r' {
			break
		} else if s.nextRune == '\\' {
			s.consumeRune()
			isEscaped = true
		} else if s.nextRune == '"' {
			s.consumeRune()
			terminated = true
		} else if !isSourceCharacter(s.nextRune) {
			s.errorf("illegal character %#U in string", s.nextRune)
			s.consumeRune()
		} else {
			value += string(s.nextRune)
			s.consumeRune()
		}
	}

	if !terminated {
		s.errorf("unterminated string")
	}

	return value
}
