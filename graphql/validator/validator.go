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
		validateDocument,
		validateOperations,
		validateFields,
		validateVariables,
	} {
		ret = append(ret, f(doc, s, typeInfo)...)
	}
	return ret
}

func unwrappedASTType(t ast.Type) *ast.NamedType {
	for {
		if t == nil {
			return nil
		}
		switch tt := t.(type) {
		case *ast.ListType:
			t = tt.Type
		case *ast.NonNullType:
			t = tt.Type
		case *ast.NamedType:
			return tt
		default:
			panic(fmt.Sprintf("unsupported ast type: %T", t))
		}
	}
}
