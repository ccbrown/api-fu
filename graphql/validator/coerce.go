package validator

import (
	"github.com/ccbrown/api-fu/graphql/ast"
	"github.com/ccbrown/api-fu/graphql/schema"
)

func CoerceVariableValues(s *schema.Schema, operation *ast.OperationDefinition, variableValues map[string]interface{}) (map[string]interface{}, *Error) {
	coercedValues := map[string]interface{}{}
	for _, def := range operation.VariableDefinitions {
		variableName := def.Variable.Name.Name
		variableType := schemaType(def.Type, s)
		if variableType == nil || !variableType.IsInputType() {
			return nil, newError(def.Type, "Invalid variable type.")
		}
		value, hasValue := variableValues[variableName]

		if !hasValue && def.DefaultValue != nil {
			coerced, err := schema.CoerceLiteral(def.DefaultValue, variableType, variableValues)
			if err != nil {
				return nil, newError(def.DefaultValue, "Invalid default value for $%v: %v", variableName, err.Error())
			}
			coercedValues[variableName] = coerced
			continue
		} else if schema.IsNonNullType(variableType) && !hasValue {
			return nil, newError(def.Variable, "The %v variable is required.", variableName)
		} else if hasValue {
			coerced, err := schema.CoerceVariableValue(value, variableType)
			if err != nil {
				return nil, newError(def.Variable, "Invalid $%v value: %v", variableName, err.Error())
			}
			coercedValues[variableName] = coerced
		}
	}
	return coercedValues, nil
}

func CoerceArgumentValues(node ast.Node, argumentDefinitions map[string]*schema.InputValueDefinition, arguments []*ast.Argument, variableValues map[string]interface{}) (map[string]interface{}, *Error) {
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
		} else if schema.IsNonNullType(argumentType) && !hasValue {
			return nil, newError(node, "The %v argument is required.", argumentName)
		} else if hasValue {
			if coercedValues == nil {
				coercedValues = map[string]interface{}{}
			}
			if argVariable, ok := argumentValue.(*ast.Variable); ok {
				coercedValues[argumentName] = variableValues[argVariable.Name.Name]
			} else if coerced, err := schema.CoerceLiteral(argumentValue, argumentType, variableValues); err != nil {
				return nil, newError(argumentValue, "Invalid argument value: %v", err.Error())
			} else {
				coercedValues[argumentName] = coerced
			}
		}
	}

	return coercedValues, nil
}
