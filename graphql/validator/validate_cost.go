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

		var coercedVariableValues map[string]interface{}
		if op != nil {
			if v, err := CoerceVariableValues(s, op, variableValues); err != nil {
				ret = append(ret, newSecondaryError(op, err.Error()))
			} else {
				coercedVariableValues = v
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
							if def, ok := typeInfo.FieldDefinitions[selection]; ok && coercedVariableValues != nil {
								if args, err := CoerceArgumentValues(selection, def.Arguments, selection.Arguments, coercedVariableValues); err != nil {
									ret = append(ret, newSecondaryError(selection, err.Error()))
								} else {
									costContext := schema.FieldCostContext{
										Context:   ctx,
										Arguments: args,
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

				if len(ret) > 0 {
					return false
				}

				multipliers = append(multipliers, newMultiplier)
				ctxs = append(ctxs, newCtx)
				return true
			})
		}

		if len(ret) == 0 && op != nil {
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
