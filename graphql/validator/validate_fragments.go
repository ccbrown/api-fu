package validator

import (
	"github.com/ccbrown/apifu/graphql/ast"
	"github.com/ccbrown/apifu/graphql/schema"
)

func validateFragments(doc *ast.Document, s *schema.Schema, typeInfo *TypeInfo) []*Error {
	return validateFragmentDeclarations(doc, s, typeInfo)
}

func validateFragmentDeclarations(doc *ast.Document, s *schema.Schema, typeInfo *TypeInfo) []*Error {
	var ret []*Error

	fragmentsByName := map[string]*ast.FragmentDefinition{}
	for _, def := range doc.Definitions {
		if def, ok := def.(*ast.FragmentDefinition); ok {
			if _, ok := fragmentsByName[def.Name.Name]; ok {
				ret = append(ret, NewError("a fragment with this name already exists"))
			} else {
				fragmentsByName[def.Name.Name] = def
			}

			switch s.NamedType(def.TypeCondition.Name.Name).(type) {
			case *schema.ObjectType, *schema.InterfaceType, *schema.UnionType:
			case nil:
				ret = append(ret, NewError("undefined type"))
			default:
				ret = append(ret, NewError("fragments may only be defined on objects, interfaces, and unions"))
			}
		}
	}

	usedFragments := map[string]struct{}{}
	ast.Inspect(doc, func(node interface{}) bool {
		switch node := node.(type) {
		case *ast.FragmentSpread:
			name := node.FragmentName.Name
			if _, ok := fragmentsByName[name]; !ok {
				ret = append(ret, NewError("undefined fragment"))
			}
			usedFragments[name] = struct{}{}
		}
		return true
	})

	for name := range fragmentsByName {
		if _, ok := usedFragments[name]; !ok {
			ret = append(ret, NewError("unused fragment"))
		}
	}

	return ret
}
