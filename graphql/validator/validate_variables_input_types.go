package validator

import (
	"github.com/ccbrown/apifu/graphql/ast"
	"github.com/ccbrown/apifu/graphql/schema"
)

func validateVariablesInputTypes(doc *ast.Document, schema *schema.Schema, typeInfo *TypeInfo) []*Error {
	var ret []*Error
	for _, def := range doc.Definitions {
		if def, ok := def.(*ast.OperationDefinition); ok {
			for _, v := range def.VariableDefinitions {
				typeName := unwrappedASTType(v.Type).Name.Name
				if t := schema.NamedType(typeName); t == nil {
					ret = append(ret, NewError("unknown type"))
				} else if !t.IsInputType() {
					ret = append(ret, NewError("%v is not an input type", typeName))
				}
			}
		}
	}
	return ret
}
