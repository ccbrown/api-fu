package validator

import (
	"github.com/ccbrown/apifu/graphql/ast"
	"github.com/ccbrown/apifu/graphql/schema"
)

func validateDocumentExecutableDefinitions(doc *ast.Document, schema *schema.Schema) []*Error {
	var ret []*Error
	for _, def := range doc.Definitions {
		switch def.(type) {
		case *ast.OperationDefinition, *ast.FragmentDefinition:
		default:
			ret = append(ret, NewError("definitions must define an operation or fragment"))
		}
	}
	return ret
}
