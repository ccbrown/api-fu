package validator

import (
	"github.com/ccbrown/api-fu/graphql/ast"
	"github.com/ccbrown/api-fu/graphql/schema"
)

func validateDocument(doc *ast.Document, schema *schema.Schema, typeInfo *TypeInfo) []*Error {
	var ret []*Error
	for _, def := range doc.Definitions {
		switch def.(type) {
		case *ast.OperationDefinition, *ast.FragmentDefinition:
		default:
			ret = append(ret, newError(def, "definitions must define an operation or fragment"))
		}
	}
	return ret
}
