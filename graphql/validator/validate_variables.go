package validator

import (
	"github.com/ccbrown/apifu/graphql/ast"
	"github.com/ccbrown/apifu/graphql/schema"
)

func validateVariables(doc *ast.Document, schema *schema.Schema, typeInfo *TypeInfo) []*Error {
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
					ret = append(ret, NewError("a variable with this name already exists"))
				} else {
					variableDefinitions[def.Variable.Name.Name] = def
				}

				if t := typeInfo.VariableDefinitionTypes[def]; t == nil {
					ret = append(ret, NewError("unknown type"))
				} else if !t.IsInputType() {
					ret = append(ret, NewError("%v is not an input type", t))
				}
			}

			encounteredVariables := map[string]struct{}{}
			unvalidatedFragmentSpreads := map[string]bool{}
			validatedFragmentSpreads := map[string]bool{}

			validate := func(node interface{}) {
				ast.Inspect(node, func(node interface{}) bool {
					switch node := node.(type) {
					case *ast.Variable:
						if def, ok := variableDefinitions[node.Name.Name]; !ok {
							ret = append(ret, NewError("undefined variable"))
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
					ret = append(ret, NewError("unused variable"))
				}
			}
		}
	}
	return ret
}

func validateVariableUsage(def *ast.VariableDefinition, usage *ast.Variable, typeInfo *TypeInfo) *Error {
	variableType := typeInfo.VariableDefinitionTypes[def]
	locationType := typeInfo.ExpectedTypes[usage]

	if variableType == nil || locationType == nil {
		return nil
	}

	if nonNullLocationType, ok := locationType.(*schema.NonNullType); ok && !schema.IsNonNullType(variableType) {
		hasNonNullVariableDefaultValue := def.DefaultValue != nil && !ast.IsNullValue(def.DefaultValue)
		hasLocationDefaultValue := typeInfo.DefaultValues[usage] != nil
		if !hasNonNullVariableDefaultValue && !hasLocationDefaultValue {
			return NewError("cannot use nullable variable where non-null type is expected")
		}
		locationType = nonNullLocationType.Type
	}

	if !areTypesCompatible(variableType, locationType) {
		return NewError("incompatible variable type")
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
