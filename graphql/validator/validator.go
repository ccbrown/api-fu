package validator

import (
	"fmt"

	"github.com/ccbrown/apifu/graphql/ast"
	"github.com/ccbrown/apifu/graphql/schema"
)

type Error struct {
	Message string
}

func (err *Error) Error() string {
	return err.Message
}

func NewError(message string, args ...interface{}) *Error {
	return &Error{
		Message: fmt.Sprintf(message, args...),
	}
}

func ValidateDocument(doc *ast.Document, s *schema.Schema) []*Error {
	typeInfo := NewTypeInfo(doc, s)
	var ret []*Error
	for _, f := range []func(*ast.Document, *schema.Schema, *TypeInfo) []*Error{
		validateDocumentExecutableDefinitions,
		validateOperationsNameUniqueness,
		validateOperationsLoneAnonymousOperation,
		validateVariablesNameUniqueness,
		validateFieldsSelectionsOnObjectsInterfacesAndUnions,
		validateFieldsLeafFieldSelections,
	} {
		ret = append(ret, f(doc, s, typeInfo)...)
	}
	return ret
}
