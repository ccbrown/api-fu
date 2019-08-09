package executor

import (
	"fmt"

	"github.com/ccbrown/api-fu/graphql/ast"
)

type Location struct {
	Line   int
	Column int
}

type Error struct {
	// Executor error messages are formatted as sentences, e.g. "An error occurred."
	Message string

	// Nearly all errors have locations, which point to one or more relevant query tokens.
	Locations []Location

	// If the error occurred during the resolution of a particular field, a path will be present.
	Path []interface{}

	originalError error
}

func (err *Error) Error() string {
	return err.Message
}

// If the error came from a resolver, you can get the original error with Unwrap.
func (err *Error) Unwrap() error {
	return err.originalError
}

func newError(node ast.Node, message string, args ...interface{}) *Error {
	return newErrorWithPath(node, nil, message, args...)
}

func newErrorWithPath(node ast.Node, path *path, message string, args ...interface{}) *Error {
	ret := &Error{
		Message: fmt.Sprintf(message, args...),
	}
	if node != nil {
		ret.Locations = []Location{{
			Line:   node.Position().Line,
			Column: node.Position().Column,
		}}
	}
	if path != nil {
		ret.Path = path.Slice()
	}
	return ret
}
