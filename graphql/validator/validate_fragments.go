package validator

import (
	"fmt"

	"github.com/ccbrown/api-fu/graphql/ast"
	"github.com/ccbrown/api-fu/graphql/schema"
)

func validateFragments(doc *ast.Document, s *schema.Schema, features schema.FeatureSet, typeInfo *TypeInfo) []*Error {
	ret := validateFragmentDeclarations(doc, s, features, typeInfo)
	ret = append(ret, validateFragmentSpreads(doc, s, features, typeInfo)...)
	return ret
}

func validateFragmentDeclarations(doc *ast.Document, s *schema.Schema, features schema.FeatureSet, typeInfo *TypeInfo) []*Error {
	var ret []*Error

	validateTypeCondition := func(tc *ast.NamedType) {
		switch namedType(s, features, tc.Name.Name).(type) {
		case *schema.ObjectType, *schema.InterfaceType, *schema.UnionType:
		case nil:
			ret = append(ret, newError(tc.Name, "undefined type"))
		default:
			ret = append(ret, newError(tc.Name, "fragments may only be defined on objects, interfaces, and unions"))
		}
	}

	fragmentsByName := map[string]*ast.FragmentDefinition{}
	for _, def := range doc.Definitions {
		if def, ok := def.(*ast.FragmentDefinition); ok {
			if _, ok := fragmentsByName[def.Name.Name]; ok {
				ret = append(ret, newError(def.Name, "a fragment with this name already exists"))
			} else {
				fragmentsByName[def.Name.Name] = def
			}
			validateTypeCondition(def.TypeCondition)
		}
	}

	usedFragments := map[string]struct{}{}
	ast.Inspect(doc, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.FragmentSpread:
			usedFragments[node.FragmentName.Name] = struct{}{}
		case *ast.InlineFragment:
			if node.TypeCondition != nil {
				validateTypeCondition(node.TypeCondition)
			}
		}
		return true
	})

	for name, def := range fragmentsByName {
		if _, ok := usedFragments[name]; !ok {
			ret = append(ret, newError(def, "unused fragment"))
		}
	}

	return ret
}

func validateFragmentSpreads(doc *ast.Document, s *schema.Schema, features schema.FeatureSet, typeInfo *TypeInfo) []*Error {
	var ret []*Error

	fragmentsByName := map[string]*ast.FragmentDefinition{}
	directFragmentDependencies := map[string]map[string]struct{}{}
	for _, def := range doc.Definitions {
		if def, ok := def.(*ast.FragmentDefinition); ok {
			fragmentsByName[def.Name.Name] = def

			deps := map[string]struct{}{}
			ast.Inspect(def, func(node ast.Node) bool {
				if node, ok := node.(*ast.FragmentSpread); ok {
					deps[node.FragmentName.Name] = struct{}{}
				}
				return true
			})
			directFragmentDependencies[def.Name.Name] = deps
		}
	}

	for name, def := range fragmentsByName {
		toVisit := []string{name}
		encountered := map[string]struct{}{}
		cycleFound := false
		for i := 0; i < len(toVisit) && !cycleFound; i++ {
			for dep := range directFragmentDependencies[toVisit[i]] {
				if _, ok := encountered[dep]; !ok {
					if dep == name {
						cycleFound = true
						break
					}
					toVisit = append(toVisit, dep)
					encountered[dep] = struct{}{}
				}
			}
		}
		if cycleFound {
			ret = append(ret, newError(def, "fragment cycle detected"))
		}
	}

	validateSpread := func(tc *ast.NamedType, parentType schema.NamedType) {
		if parentType == nil {
			ret = append(ret, newSecondaryError(tc, "no type info for fragment spread parent"))
			return
		}
		switch fragmentType := namedType(s, features, tc.Name.Name).(type) {
		case *schema.ObjectType, *schema.InterfaceType, *schema.UnionType:
			a := getPossibleTypes(s, fragmentType)
			b := getPossibleTypes(s, parentType)
			hasIntersection := false
			for k := range a {
				if _, ok := b[k]; ok {
					hasIntersection = true
					break
				}
			}
			if !hasIntersection {
				ret = append(ret, newError(tc, "impossible fragment spread"))
			}
		default:
		}
	}

	var selectionSetTypes []schema.NamedType
	ast.Inspect(doc, func(node ast.Node) bool {
		if node == nil {
			selectionSetTypes = selectionSetTypes[:len(selectionSetTypes)-1]
			return true
		}

		var selectionSetType schema.NamedType
		switch node := node.(type) {
		case *ast.SelectionSet:
			selectionSetType = typeInfo.SelectionSetTypes[node]
		case *ast.FragmentSpread:
			name := node.FragmentName.Name
			if def, ok := fragmentsByName[name]; !ok {
				ret = append(ret, newError(node.FragmentName, "undefined fragment"))
			} else {
				validateSpread(def.TypeCondition, selectionSetTypes[len(selectionSetTypes)-1])
			}
		case *ast.InlineFragment:
			if node.TypeCondition != nil {
				validateSpread(node.TypeCondition, selectionSetTypes[len(selectionSetTypes)-1])
			}
		}
		selectionSetTypes = append(selectionSetTypes, selectionSetType)
		return true
	})

	return ret
}

func getPossibleTypes(s *schema.Schema, t schema.NamedType) map[string]schema.NamedType {
	ret := map[string]schema.NamedType{}
	switch t := t.(type) {
	case *schema.ObjectType:
		ret[t.Name] = t
	case *schema.InterfaceType:
		for _, obj := range s.InterfaceImplementations(t.Name) {
			ret[obj.Name] = obj
		}
	case *schema.UnionType:
		for _, t := range t.MemberTypes {
			ret[t.TypeName()] = t
		}
	default:
		panic(fmt.Sprintf("unexpected type: %T", t))
	}
	return ret
}
