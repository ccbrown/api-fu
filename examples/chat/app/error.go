package app

type SanitizedError interface {
	error
	SanitizedError() string
}

type InternalError struct {
	cause error
}

func (e *InternalError) Error() string {
	return "An internal error has occurred."
}

func (e *InternalError) SanitizedError() string {
	return e.Error()
}

func (e *InternalError) Unwrap() error {
	return e.cause
}

func (s *Session) InternalError(err error) *InternalError {
	s.Logger.Error(err)
	return &InternalError{
		cause: err,
	}
}

type UserError struct {
	message string
}

func (e *UserError) Error() string {
	return e.message
}

func (e *UserError) SanitizedError() string {
	return e.Error()
}

func (s *Session) UserError(message string) *UserError {
	return &UserError{
		message: message,
	}
}
