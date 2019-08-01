package validator

import (
	"github.com/ccbrown/apifu/graphql/ast"
	"github.com/ccbrown/apifu/graphql/schema"
)

func validateVariablesNameUniqueness(doc *ast.Document, schema *schema.Schema) []*Error {
	var ret []*Error
	for _, def := range doc.Definitions {
		if def, ok := def.(*ast.OperationDefinition); ok {
			variableNames := map[string]struct{}{}
			for _, v := range def.VariableDefinitions {
				name := v.Variable.Name.Name
				if _, ok := variableNames[name]; ok {
					ret = append(ret, NewError("a variable with this name already exists"))
				} else {
					variableNames[name] = struct{}{}
				}
			}
		}
	}
	return ret
}
