package validator

import (
	"github.com/ccbrown/api-fu/graphql/ast"
	"github.com/ccbrown/api-fu/graphql/schema"
)

func validateOperations(doc *ast.Document, schema *schema.Schema, features schema.FeatureSet, typeInfo *TypeInfo) []*Error {
	var ret []*Error

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
			if def.Name == nil {
				anonymousOperationCount++
			} else if _, ok := operationNames[def.Name.Name]; ok {
				ret = append(ret, newError(def.Name, "an operation with this name already exists"))
			} else {
				operationNames[def.Name.Name] = struct{}{}
			}

			if _, ok := typeInfo.SelectionSetTypes[def.SelectionSet]; !ok {
				ret = append(ret, newError(def, "unsupported operation type"))
			}

			if opType := def.OperationType; opType != nil && opType.Value == "subscription" {
				fieldsForName := map[string][]fieldAndParent{}
				if err := addFieldSelections(fieldsForName, def.SelectionSet, fragmentDefinitions); err != nil {
					ret = append(ret, err)
				} else if len(fieldsForName) != 1 {
					ret = append(ret, newError(def, "subscriptions may only have one root field"))
				}
			}
		}
	}

	if anonymousOperationCount > 0 {
		seen := 0
		for _, def := range doc.Definitions {
			if def, ok := def.(*ast.OperationDefinition); ok {
				seen++
				if seen == 2 {
					ret = append(ret, newError(def, "only one operation is allowed when an anonymous operation is present"))
					break
				}
			}
		}
	}

	return ret
}
