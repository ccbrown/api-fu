package validator

import (
	"github.com/ccbrown/apifu/graphql/ast"
	"github.com/ccbrown/apifu/graphql/schema"
)

func validateOperations(doc *ast.Document, schema *schema.Schema, typeInfo *TypeInfo) []*Error {
	var ret []*Error

	operationCount := 0
	anonymousOperationCount := 0
	operationNames := map[string]struct{}{}

	for _, def := range doc.Definitions {
		if def, ok := def.(*ast.OperationDefinition); ok {
			operationCount++
			if def.Name == nil {
				anonymousOperationCount++
			} else if _, ok := operationNames[def.Name.Name]; ok {
				ret = append(ret, NewError("an operation with this name already exists"))
			} else {
				operationNames[def.Name.Name] = struct{}{}
			}
		}
	}

	if operationCount > 1 && anonymousOperationCount > 0 {
		ret = append(ret, NewError("only one operation is allowed when an anonymous operation is present"))
	}
	return ret
}
