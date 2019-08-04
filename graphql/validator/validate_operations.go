package validator

import (
	"github.com/ccbrown/api-fu/graphql/ast"
	"github.com/ccbrown/api-fu/graphql/schema"
)

func validateOperations(doc *ast.Document, schema *schema.Schema, typeInfo *TypeInfo) []*Error {
	var ret []*Error

	operationCount := 0
	anonymousOperationCount := 0
	operationNames := map[string]struct{}{}

	fragmentDefinitions := map[string]*ast.FragmentDefinition{}
	for _, def := range doc.Definitions {
		if def, ok := def.(*ast.FragmentDefinition); ok {
			fragmentDefinitions[def.Name.Name] = def
		}
	}

	for _, def := range doc.Definitions {
		if def, ok := def.(*ast.OperationDefinition); ok {
			operationCount++
			if def.Name == nil {
				anonymousOperationCount++
			} else if _, ok := operationNames[def.Name.Name]; ok {
				ret = append(ret, newError("an operation with this name already exists"))
			} else {
				operationNames[def.Name.Name] = struct{}{}
			}

			if def.OperationType != nil && *def.OperationType == ast.OperationTypeSubscription {
				fieldsForName := map[string][]fieldAndParent{}
				if err := addFieldSelections(fieldsForName, def.SelectionSet, fragmentDefinitions); err != nil {
					ret = append(ret, err)
				} else if len(fieldsForName) != 1 {
					ret = append(ret, newError("subscriptions may only have one root field"))
				}
			}
		}
	}

	if operationCount > 1 && anonymousOperationCount > 0 {
		ret = append(ret, newError("only one operation is allowed when an anonymous operation is present"))
	}
	return ret
}
