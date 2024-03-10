package validator

import (
	"github.com/ccbrown/api-fu/graphql/ast"
	"github.com/ccbrown/api-fu/graphql/schema"
)

func validateVariables(doc *ast.Document, schema *schema.Schema, features schema.FeatureSet, typeInfo *TypeInfo) []*Error {
	fragmentDefinitions := map[string]*ast.FragmentDefinition{}
	for _, def := range doc.Definitions {
		if def, ok := def.(*ast.FragmentDefinition); ok {
			fragmentDefinitions[def.Name.Name] = def
		}
	}

	var ret []*Error
	for _, def := range doc.Definitions {
		if def, ok := def.(*ast.OperationDefinition); ok {
			variableDefinitions := map[string]*ast.VariableDefinition{}
			for _, def := range def.VariableDefinitions {
				name := def.Variable.Name.Name
				if _, ok := variableDefinitions[name]; ok {
					ret = append(ret, newError(def.Variable.Name, "a variable with this name already exists"))
				} else {
					variableDefinitions[def.Variable.Name.Name] = def
				}

				if t := typeInfo.VariableDefinitionTypes[def]; t == nil {
					ret = append(ret, newError(def.Type, "unknown type"))
				} else if !t.IsInputType() {
					ret = append(ret, newError(def.Type, "%v is not an input type", t))
				}
			}

			encounteredVariables := map[string]struct{}{}
			unvalidatedFragmentSpreads := map[string]bool{}
			validatedFragmentSpreads := map[string]bool{}

			validate := func(node ast.Node) {
				ast.Inspect(node, func(node ast.Node) bool {
					switch node := node.(type) {
					case *ast.Variable:
						if def, ok := variableDefinitions[node.Name.Name]; !ok {
							ret = append(ret, newError(node, "undefined variable"))
						} else if err := validateVariableUsage(def, node, typeInfo); err != nil {
							ret = append(ret, err)
						}
						encounteredVariables[node.Name.Name] = struct{}{}
					case *ast.VariableDefinition:
						return false
					case *ast.FragmentSpread:
						if name := node.FragmentName.Name; !validatedFragmentSpreads[name] {
							unvalidatedFragmentSpreads[name] = true
						}
					}
					return true
				})
			}
			validate(def)

			for len(unvalidatedFragmentSpreads) > 0 {
				for name := range unvalidatedFragmentSpreads {
					delete(unvalidatedFragmentSpreads, name)
					validatedFragmentSpreads[name] = true
					if def, ok := fragmentDefinitions[name]; ok {
						validate(def)
					}
				}
			}

			for _, v := range def.VariableDefinitions {
				if _, ok := encounteredVariables[v.Variable.Name.Name]; !ok {
					ret = append(ret, newError(v.Variable, "unused variable"))
				}
			}
		}
	}
	return ret
}

func validateVariableUsage(def *ast.VariableDefinition, usage *ast.Variable, typeInfo *TypeInfo) *Error {
	variableType := typeInfo.VariableDefinitionTypes[def]
	locationType := typeInfo.ExpectedTypes[usage]

	if variableType == nil {
		return newSecondaryError(def, "no type info for variable type")
	} else if locationType == nil {
		return newSecondaryError(usage, "no type info for location type")
	}

	if nonNullLocationType, ok := locationType.(*schema.NonNullType); ok && !schema.IsNonNullType(variableType) {
		hasNonNullVariableDefaultValue := def.DefaultValue != nil && !ast.IsNullValue(def.DefaultValue)
		hasLocationDefaultValue := typeInfo.DefaultValues[usage] != nil
		if !hasNonNullVariableDefaultValue && !hasLocationDefaultValue {
			return newError(usage, "cannot use nullable variable where non-null type is expected")
		}
		locationType = nonNullLocationType.Type
	}

	if !areTypesCompatible(variableType, locationType) {
		return newError(usage, "incompatible variable type")
	}

	return nil
}

func areTypesCompatible(variableType, locationType schema.Type) bool {
	if nonNullLocationType, ok := locationType.(*schema.NonNullType); ok {
		if nonNullVariableType, ok := variableType.(*schema.NonNullType); ok {
			return areTypesCompatible(nonNullVariableType.Type, nonNullLocationType.Type)
		}
		return false
	}

	if nonNullVariableType, ok := variableType.(*schema.NonNullType); ok {
		return areTypesCompatible(nonNullVariableType.Type, locationType)
	}

	if listLocationType, ok := locationType.(*schema.ListType); ok {
		if listVariableType, ok := variableType.(*schema.ListType); ok {
			return areTypesCompatible(listVariableType.Type, listLocationType.Type)
		}
		return false
	}

	if _, ok := variableType.(*schema.ListType); ok {
		return false
	}

	return variableType.IsSameType(locationType)
}
