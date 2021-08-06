package validator

import (
	"context"

	"github.com/ccbrown/api-fu/graphql/ast"
	"github.com/ccbrown/api-fu/graphql/schema"
)

const maxUint = ^uint(0)
const minUint = 0
const maxInt = int(maxUint >> 1)
const minInt = -maxInt - 1

// Multiplies two non-negative numbers, returning -1 if either is negative or if they would
// overflow.
func checkedNonNegativeMultiply(a, b int) int {
	if a < 0 || b < 0 {
		return -1
	} else if a == 0 || b == 0 || a == 1 || b == 1 {
		return a * b
	}
	c := a * b
	if c/b != a {
		return -1
	}
	return c
}

// Adds two non-negative numbers, returning -1 if either is negative or if they would overflow.
func checkedNonNegativeAdd(a, b int) int {
	if a < 0 || b < 0 || a > maxInt-b {
		return -1
	}
	return a + b
}

// Calculates the cost of the given operation and ensures it is not greater than max. If max is -1,
// no limit is enforced. If actual is non-nil, it is set to the actual cost of the operation.
// Queries with costs that are too high to calculate due to overflows always result in an error when
// max is non-negative, and actual will be set to the maximum possible value.
func ValidateCost(operationName string, variableValues map[string]interface{}, max int, actual *int, defaultCost schema.FieldCost) Rule {
	return func(doc *ast.Document, s *schema.Schema, typeInfo *TypeInfo) []*Error {
		var ret []*Error

		var op *ast.OperationDefinition
		for _, def := range doc.Definitions {
			if def, ok := def.(*ast.OperationDefinition); ok {
				if operationName == "" || (def.Name != nil && def.Name.Name == operationName) {
					if op != nil {
						op = nil
						break
					}
					op = def
				}
			}
		}

		fragmentsByName := map[string]*ast.FragmentDefinition{}
		for _, def := range doc.Definitions {
			if def, ok := def.(*ast.FragmentDefinition); ok {
				fragmentsByName[def.Name.Name] = def
			}
		}

		var cost int
		multipliers := []int{1}
		ctxs := []context.Context{context.Background()}
		fragments := map[string]struct{}{}

		var visitNode func(node ast.Node)
		visitNode = func(node ast.Node) {
			ast.Inspect(node, func(node ast.Node) bool {
				if node == nil {
					multipliers = multipliers[:len(multipliers)-1]
					ctxs = ctxs[:len(ctxs)-1]
				}

				multiplier := multipliers[len(multipliers)-1]
				ctx := ctxs[len(ctxs)-1]
				newMultiplier := multiplier
				newCtx := ctx

				if selectionSet, ok := node.(*ast.SelectionSet); ok {
					for _, selection := range selectionSet.Selections {
						switch selection := selection.(type) {
						case *ast.Field:
							if def, ok := typeInfo.FieldDefinitions[selection]; ok {
								costContext := schema.FieldCostContext{
									Context:   ctx,
									Arguments: coerceArgumentValues(selection, def.Arguments, selection.Arguments, variableValues),
								}
								fieldCost := defaultCost
								if def.Cost != nil {
									fieldCost = def.Cost(&costContext)
								}
								cost = checkedNonNegativeAdd(cost, checkedNonNegativeMultiply(multiplier, fieldCost.Resolver))
								if fieldCost.Multiplier > 1 {
									newMultiplier = checkedNonNegativeMultiply(multiplier, fieldCost.Multiplier)
								}
								if fieldCost.Context != nil {
									newCtx = fieldCost.Context
								}
							} else if selection.Name.Name != "__typename" {
								ret = append(ret, newSecondaryError(selection, "unknown field type"))
							}
						case *ast.FragmentSpread:
							if _, ok := fragments[selection.FragmentName.Name]; ok {
								ret = append(ret, newSecondaryError(selection, "fragment cycle detected"))
							} else if def, ok := fragmentsByName[selection.FragmentName.Name]; ok {
								fragments[selection.FragmentName.Name] = struct{}{}
								visitNode(def)
								delete(fragments, selection.FragmentName.Name)
							} else {
								ret = append(ret, newSecondaryError(selection, "undefined fragment"))
							}
						}
					}
				}

				multipliers = append(multipliers, newMultiplier)
				ctxs = append(ctxs, newCtx)
				return true
			})
		}

		if op != nil {
			visitNode(op)
		}

		if len(ret) == 0 {
			if actual != nil {
				if cost < 0 {
					*actual = maxInt
				} else {
					*actual = cost
				}
			}

			if max >= 0 {
				if cost < 0 {
					ret = append(ret, newError(op, "operation cost is too high to calculate"))
				} else if cost > max {
					ret = append(ret, newError(op, "operation cost of %v exceeds allowed cost of %v", cost, max))
				}
			}
		}

		return ret
	}
}

// Makes a best effort attempt to coerce argument values, ignoring any errors that occur.
func coerceArgumentValues(node ast.Node, argumentDefinitions map[string]*schema.InputValueDefinition, arguments []*ast.Argument, variableValues map[string]interface{}) map[string]interface{} {
	var coercedValues map[string]interface{}

	argumentValues := map[string]ast.Value{}
	for _, arg := range arguments {
		argumentValues[arg.Name.Name] = arg.Value
	}

	for argumentName, argumentDefinition := range argumentDefinitions {
		argumentType := argumentDefinition.Type
		defaultValue := argumentDefinition.DefaultValue

		argumentValue, hasValue := argumentValues[argumentName]

		if argumentValue, ok := argumentValue.(*ast.Variable); ok {
			_, hasValue = variableValues[argumentValue.Name.Name]
		}

		if !hasValue && defaultValue != nil {
			if defaultValue == schema.Null {
				defaultValue = nil
			}
			if coercedValues == nil {
				coercedValues = map[string]interface{}{}
			}
			coercedValues[argumentName] = defaultValue
		} else if hasValue {
			if coercedValues == nil {
				coercedValues = map[string]interface{}{}
			}
			if argVariable, ok := argumentValue.(*ast.Variable); ok {
				coercedValues[argumentName] = variableValues[argVariable.Name.Name]
			} else if coerced, err := schema.CoerceLiteral(argumentValue, argumentType, variableValues); err == nil {
				coercedValues[argumentName] = coerced
			}
		}
	}

	return coercedValues
}
