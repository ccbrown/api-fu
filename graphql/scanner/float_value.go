package scanner

func (s *Scanner) consumeFractionalPart() bool {
	if s.nextRune != '.' || !isDigit(s.peek()) {
		return false
	}
	s.consumeRune()
	for !s.isDone() && isDigit(s.nextRune) {
		s.consumeRune()
	}
	return true
}

func (s *Scanner) consumeExponentPart() bool {
	if s.nextRune != 'e' && s.nextRune != 'E' {
		return false
	}
	s.consumeRune()
	if s.nextRune == '+' || s.nextRune == '-' {
		s.consumeRune()
	}
	if !isDigit(s.nextRune) {
		s.errorf("exponent digit expected")
	}
	for !s.isDone() && isDigit(s.nextRune) {
		s.consumeRune()
	}
	return true
}
