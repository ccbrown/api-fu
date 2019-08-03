package validator

import (
	"github.com/ccbrown/apifu/graphql/ast"
	"github.com/ccbrown/apifu/graphql/schema"
)

func validateDocument(doc *ast.Document, schema *schema.Schema, typeInfo *TypeInfo) []*Error {
	var ret []*Error
	for _, def := range doc.Definitions {
		switch def.(type) {
		case *ast.OperationDefinition, *ast.FragmentDefinition:
		default:
			ret = append(ret, newError("definitions must define an operation or fragment"))
		}
	}
	return ret
}
