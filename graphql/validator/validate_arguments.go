package validator

import (
	"github.com/ccbrown/api-fu/graphql/ast"
	"github.com/ccbrown/api-fu/graphql/schema"
)

func validateArguments(doc *ast.Document, s *schema.Schema, typeInfo *TypeInfo) []*Error {
	var ret []*Error
	ast.Inspect(doc, func(node interface{}) bool {
		var arguments []*ast.Argument
		var argumentDefinitions map[string]*schema.InputValueDefinition

		switch node := node.(type) {
		case *ast.Directive:
			if def := s.DirectiveDefinition(node.Name.Name); def != nil {
				arguments = node.Arguments
				argumentDefinitions = def.Arguments
			} else {
				ret = append(ret, newSecondaryError("undefined directive"))
				return false
			}
		case *ast.Field:
			arguments = node.Arguments
			if def := typeInfo.FieldDefinitions[node]; def != nil {
				argumentDefinitions = def.Arguments
			} else if node.Name.Name != "__typename" {
				ret = append(ret, newSecondaryError("no type info for field"))
				return false
			}
		case *ast.Argument:
			ret = append(ret, newError("unsupported argument location"))
		}

		if len(arguments) == 0 && len(argumentDefinitions) == 0 {
			return true
		}

		argumentsByName := map[string]*ast.Argument{}
		for _, argument := range arguments {
			name := argument.Name.Name
			if def := argumentDefinitions[name]; def == nil {
				ret = append(ret, newError("undefined argument"))
			} else if _, ok := argumentsByName[name]; ok {
				ret = append(ret, newError("argument already exists at this location"))
			} else {
				argumentsByName[name] = argument
			}
		}

		for name, def := range argumentDefinitions {
			if schema.IsNonNullType(def.Type) && def.DefaultValue == nil {
				if arg, ok := argumentsByName[name]; !ok {
					ret = append(ret, newError("the %v argument is required", name))
				} else if _, ok := arg.Value.(*ast.NullValue); ok {
					ret = append(ret, newError("the %v argument cannot be null", name))
				}
			}
		}

		return false
	})
	return ret
}
