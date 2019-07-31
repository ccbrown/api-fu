package scanner

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func (s *Scanner) consumeIntegerPart() bool {
	if s.nextRune == '-' && isDigit(s.peek()) {
		s.consumeRune()
	}

	if s.nextRune == '0' {
		s.consumeRune()
		return true
	} else if !isDigit(s.nextRune) {
		return false
	}

	for !s.isDone() && isDigit(s.nextRune) {
		s.consumeRune()
	}
	return true
}
