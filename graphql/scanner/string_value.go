package scanner

import "strings"

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

func blockStringValue(rawValue string) string {
	rawValue = strings.ReplaceAll(rawValue, "\r\n", "\n")
	rawValue = strings.ReplaceAll(rawValue, "\r", "\n")
	lines := strings.Split(rawValue, "\n")

	commonIndent := -1
	for _, line := range lines[1:] {
		indent := 0
		for _, r := range line {
			if r != ' ' && r != '\t' {
				break
			}
			indent++
		}
		if indent < len(line) && (commonIndent == -1 || indent < commonIndent) {
			commonIndent = indent
		}
	}

	if commonIndent > 0 {
		for i, line := range lines {
			if i > 0 && len(line) >= commonIndent {
				lines[i] = line[commonIndent:]
			}
		}
	}

	for len(lines) > 0 {
		if strings.IndexFunc(lines[0], func(r rune) bool { return r != ' ' && r != '\t' }) == -1 {
			lines = lines[1:]
		} else if len(lines) > 1 && strings.IndexFunc(lines[len(lines)-1], func(r rune) bool { return r != ' ' && r != '\t' }) == -1 {
			lines = lines[:len(lines)-1]
		} else {
			break
		}
	}

	return strings.Join(lines, "\n")
}

func (s *Scanner) consumeStringValue() string {
	s.consumeRune() // '"'

	isBlock := false
	if s.nextRune == '"' && s.peek() == '"' {
		s.consumeRune()
		s.consumeRune()
		isBlock = true
	}

	value := ""

	terminated := false
	isEscaped := false
	for !terminated && !s.isDone() {
		if isEscaped {
			if isBlock {
				if r := s.consumeRune(); r == '"' && s.nextRune == '"' && s.peek() == '"' {
					s.consumeRune()
					s.consumeRune()
					value += `"""`
				} else {
					value += string(`\`) + string(r)
				}
			} else {
				switch r := s.consumeRune(); r {
				case '"', '\\', '/':
					value += string(r)
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
			}
			isEscaped = false
			continue
		}

		if s.nextRune == '\n' || s.nextRune == '\r' {
			if !isBlock {
				break
			}
			value += string(s.nextRune)
			if s.consumeRune() == '\r' && s.nextRune == '\n' {
				value += string(s.consumeRune())
			}
		} else if s.nextRune == '\\' {
			s.consumeRune()
			isEscaped = true
		} else if s.nextRune == '"' {
			s.consumeRune()
			if isBlock {
				if s.nextRune == '"' && s.peek() == '"' {
					s.consumeRune()
					s.consumeRune()
					terminated = true
				} else {
					value += `"`
				}
			} else {
				terminated = true
			}
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

	if isBlock {
		value = blockStringValue(value)
	}

	return value
}
