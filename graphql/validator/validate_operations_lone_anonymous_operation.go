package validator

import (
	"github.com/ccbrown/go-api/graphql/ast"
	"github.com/ccbrown/go-api/graphql/schema"
)

func validateOperationsLoneAnonymousOperation(doc *ast.Document, schema *schema.Schema) []*Error {
	operationCount := 0
	anonymousOperationCount := 0
	for _, def := range doc.Definitions {
		if def, ok := def.(*ast.OperationDefinition); ok {
			operationCount++
			if def.Name == nil {
				anonymousOperationCount++
			}
			if operationCount > 1 && anonymousOperationCount > 0 {
				return []*Error{NewError("only one operation is allowed when an anonymous operation is present")}
			}
		}
	}
	return nil
}
