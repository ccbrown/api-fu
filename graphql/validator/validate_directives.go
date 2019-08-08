package validator

import (
	"github.com/ccbrown/api-fu/graphql/ast"
	"github.com/ccbrown/api-fu/graphql/schema"
)

func validateDirectives(doc *ast.Document, s *schema.Schema, typeInfo *TypeInfo) []*Error {
	var ret []*Error
	ast.Inspect(doc, func(node ast.Node) bool {
		var directives []*ast.Directive
		var location schema.DirectiveLocation

		switch node := node.(type) {
		case *ast.OperationDefinition:
			directives = node.Directives
			if op := node.OperationType; op == nil || op.Value == "query" {
				location = schema.DirectiveLocationQuery
			} else if op.Value == "mutation" {
				location = schema.DirectiveLocationMutation
			} else if op.Value == "subscription" {
				location = schema.DirectiveLocationSubscription
			}
		case *ast.FragmentDefinition:
			directives = node.Directives
			location = schema.DirectiveLocationFragmentDefinition
		case *ast.Field:
			directives = node.Directives
			location = schema.DirectiveLocationField
		case *ast.FragmentSpread:
			directives = node.Directives
			location = schema.DirectiveLocationFragmentSpread
		case *ast.InlineFragment:
			directives = node.Directives
			location = schema.DirectiveLocationInlineFragment
		case *ast.Directive:
			ret = append(ret, newError("unsupported directive location"))
		}

		if len(directives) == 0 {
			return true
		}

		directiveNames := map[string]struct{}{}
		for _, directive := range directives {
			name := directive.Name.Name

			if def := s.DirectiveDefinition(name); def == nil {
				ret = append(ret, newError("undefined directive"))
			} else {
				allowedLocation := false
				for _, allowed := range def.Locations {
					if allowed == location {
						allowedLocation = true
						break
					}
				}
				if !allowedLocation {
					ret = append(ret, newError("this directive is not allowed at this location"))
				}
			}

			if _, ok := directiveNames[name]; ok {
				ret = append(ret, newError("directive already exists at this location"))
			} else {
				directiveNames[name] = struct{}{}
			}
		}
		return false
	})
	return ret
}
