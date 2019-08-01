package validator

import (
	"github.com/ccbrown/apifu/graphql/ast"
	"github.com/ccbrown/apifu/graphql/schema"
)

type Error struct {
	Message string
}

func (err *Error) Error() string {
	return err.Message
}

func NewError(message string) *Error {
	return &Error{
		Message: message,
	}
}

func ValidateDocument(doc *ast.Document, s *schema.Schema) []*Error {
	var ret []*Error
	for _, f := range []func(*ast.Document, *schema.Schema) []*Error{
		validateDocumentExecutableDefinitions,
		validateOperationsNameUniqueness,
		validateOperationsLoneAnonymousOperation,
		validateVariablesNameUniqueness,
	} {
		ret = append(ret, f(doc, s)...)
	}
	return ret
}
