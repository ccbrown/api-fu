package validator

import (
	"github.com/ccbrown/apifu/graphql/ast"
	"github.com/ccbrown/apifu/graphql/schema"
)

func validateOperationsNameUniqueness(doc *ast.Document, schema *schema.Schema) []*Error {
	var ret []*Error
	operationNames := map[string]struct{}{}
	for _, def := range doc.Definitions {
		if def, ok := def.(*ast.OperationDefinition); ok && def.Name != nil {
			if _, ok := operationNames[def.Name.Name]; ok {
				ret = append(ret, NewError("an operation with this name already exists"))
			} else {
				operationNames[def.Name.Name] = struct{}{}
			}
		}
	}
	return ret
}
