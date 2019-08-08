package executor

import (
	"fmt"
	"reflect"

	"github.com/ccbrown/api-fu/graphql/ast"
	"github.com/ccbrown/api-fu/graphql/schema"
)

type Location struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

type Error struct {
	Message   string        `json:"message"`
	Locations []Location    `json:"locations,omitempty"`
	Path      []interface{} `json:"path,omitempty"`
}

func (err *Error) Error() string {
	return err.Message
}

func newError(message string, args ...interface{}) *Error {
	return &Error{
		Message: fmt.Sprintf(message, args...),
	}
}

type Request struct {
	Document       *ast.Document
	Schema         *schema.Schema
	OperationName  string
	VariableValues map[string]interface{}
	InitialValue   interface{}
}

func ExecuteRequest(r *Request) (*OrderedMap, []*Error) {
	operation, err := getOperation(r.Document, r.OperationName)
	if err != nil {
		return nil, []*Error{err}
	}
	coercedVariableValues, err := coerceVariableValues(r.Schema, operation, r.VariableValues)
	if err != nil {
		return nil, []*Error{err}
	}

	e := &executor{
		Schema:              r.Schema,
		FragmentDefinitions: map[string]*ast.FragmentDefinition{},
		VariableValues:      coercedVariableValues,
	}
	for _, def := range r.Document.Definitions {
		if def, ok := def.(*ast.FragmentDefinition); ok {
			e.FragmentDefinitions[def.Name.Name] = def
		}
	}

	if operation.OperationType == nil || operation.OperationType.Value == "query" {
		return e.executeQuery(operation, r.InitialValue)
	} else if operation.OperationType.Value == "mutation" {
		return e.executeMutation(operation, r.InitialValue)
	} else if operation.OperationType.Value == "subscription" {
		return e.subscribe(operation, r.InitialValue)
	}
	panic("unexpected operation type")
}

type executor struct {
	Schema              *schema.Schema
	FragmentDefinitions map[string]*ast.FragmentDefinition
	VariableValues      map[string]interface{}
	Errors              []*Error
}

func (e *executor) executeQuery(query *ast.OperationDefinition, initialValue interface{}) (*OrderedMap, []*Error) {
	queryType := e.Schema.QueryType()
	if !schema.IsObjectType(queryType) {
		return nil, []*Error{newError("This schema cannot perform queries.")}
	}
	data, err := e.executeSelections(query.SelectionSet.Selections, queryType, initialValue, nil, false)
	if err != nil {
		e.Errors = append(e.Errors, newError("%v", err.Error()))
	}
	return data, e.Errors
}

func (e *executor) executeMutation(mutation *ast.OperationDefinition, initialValue interface{}) (*OrderedMap, []*Error) {
	mutationType := e.Schema.MutationType()
	if !schema.IsObjectType(mutationType) {
		return nil, []*Error{newError("This schema cannot perform mutations.")}
	}
	data, err := e.executeSelections(mutation.SelectionSet.Selections, mutationType, initialValue, nil, true)
	if err != nil {
		e.Errors = append(e.Errors, newError("%v", err.Error()))
	}
	return data, e.Errors
}

func (e *executor) subscribe(subscription *ast.OperationDefinition, initialValue interface{}) (*OrderedMap, []*Error) {
	// TODO: event stream api

	subscriptionType := e.Schema.SubscriptionType()
	if !schema.IsObjectType(subscriptionType) {
		return nil, []*Error{newError("This schema cannot perform subscriptions.")}
	}
	data, err := e.executeSelections(subscription.SelectionSet.Selections, subscriptionType, initialValue, nil, false)
	if err != nil {
		e.Errors = append(e.Errors, newError("%v", err.Error()))
	}
	return data, e.Errors
}

func (e *executor) executeSelections(selections []ast.Selection, objectType *schema.ObjectType, objectValue interface{}, path []interface{}, forceSerial bool) (*OrderedMap, error) {
	// TODO: parallel execution

	groupedFieldSet := NewOrderedMap()
	e.collectFields(objectType, selections, nil, groupedFieldSet)

	resultMap := NewOrderedMap()
	for _, responseKey := range groupedFieldSet.Keys() {
		v, _ := groupedFieldSet.Get(responseKey)
		fields := v.([]*ast.Field)

		if fieldName := fields[0].Name.Name; fieldName == "__typename" {
			resultMap.Set(responseKey, objectType.Name)
		} else if fieldDef := objectType.Fields[fieldName]; fieldDef != nil {
			fieldPath := append(path, responseKey)
			responseValue, err := e.executeField(objectType, objectValue, fields, fieldDef.Type, fieldPath)
			if err != nil {
				var responseError *Error
				switch err := err.(type) {
				case *Error:
					responseError = err
				default:
					locations := make([]Location, len(fields))
					for i, field := range fields {
						locations[i].Line = field.Position().Line
						locations[i].Column = field.Position().Column
					}
					responseError = &Error{
						Message:   err.Error(),
						Locations: locations,
						Path:      fieldPath,
					}
				}

				if schema.IsNonNullType(fieldDef.Type) {
					return nil, responseError
				} else {
					e.Errors = append(e.Errors, responseError)
				}
			}
			resultMap.Set(responseKey, responseValue)
		}
	}
	return resultMap, nil
}

func isNil(v interface{}) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	return (rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface) && rv.IsNil()
}

func (e *executor) executeField(objectType *schema.ObjectType, objectValue interface{}, fields []*ast.Field, fieldType schema.Type, path []interface{}) (interface{}, error) {
	field := fields[0]
	fieldName := field.Name.Name
	fieldDef := objectType.Fields[fieldName]
	if fieldDef == nil {
		return nil, fmt.Errorf("undefined field")
	}
	argumentValues, err := coerceArgumentValues(fieldDef.Arguments, field.Arguments, e.VariableValues)
	if err != nil {
		return nil, err
	}
	resolvedValue, err := fieldDef.Resolve(&schema.FieldContext{
		Object:    objectValue,
		Arguments: argumentValues,
	})
	if !isNil(err) {
		return nil, err
	}
	return e.completeValue(fieldType, fields, resolvedValue, path)
}

func (e *executor) completeValue(fieldType schema.Type, fields []*ast.Field, result interface{}, path []interface{}) (interface{}, error) {
	if nonNullType, ok := fieldType.(*schema.NonNullType); ok {
		completedResult, err := e.completeValue(nonNullType.Type, fields, result, path)
		if err != nil {
			return nil, err
		} else if completedResult == nil {
			return nil, fmt.Errorf("null result for non-null field")
		}
		return completedResult, nil
	}

	if isNil(result) {
		return nil, nil
	}

	switch fieldType := fieldType.(type) {
	case *schema.ListType:
		result := reflect.ValueOf(result)
		if result.Kind() != reflect.Slice {
			return nil, fmt.Errorf("result is not a list")
		}
		innerType := fieldType.Type
		completedResult := make([]interface{}, result.Len())
		for i := range completedResult {
			completedResultItem, err := e.completeValue(innerType, fields, result.Index(i).Interface(), append(path, i))
			if err != nil {
				return nil, err
			}
			completedResult[i] = completedResultItem
		}
		return completedResult, nil
	case *schema.ScalarType:
		return fieldType.CoerceResult(result)
	case *schema.EnumType:
		return fieldType.CoerceResult(result)
	case *schema.ObjectType, *schema.InterfaceType, *schema.UnionType:
		var objectType *schema.ObjectType
		switch fieldType := fieldType.(type) {
		case *schema.ObjectType:
			objectType = fieldType
		case *schema.InterfaceType:
			objectType = fieldType.ObjectType(result)
		case *schema.UnionType:
			objectType = fieldType.ObjectType(result)
		}
		if objectType == nil {
			return nil, fmt.Errorf("unknown object type")
		}
		return e.executeSelections(mergeSelectionSets(fields), objectType, result, path, false)
	}
	panic(fmt.Sprintf("unexpected field type: %T", fieldType))
}

func mergeSelectionSets(fields []*ast.Field) []ast.Selection {
	var selectionSet []ast.Selection
	for _, field := range fields {
		if field.SelectionSet == nil {
			continue
		}
		selectionSet = append(selectionSet, field.SelectionSet.Selections...)
	}
	return selectionSet
}

func (e *executor) collectFields(objectType *schema.ObjectType, selections []ast.Selection, visitedFragments map[string]struct{}, groupedFields *OrderedMap) {
	if visitedFragments == nil {
		visitedFragments = map[string]struct{}{}
	}
	for _, selection := range selections {
		skip := false
		for _, directive := range selection.SelectionDirectives() {
			if def := e.Schema.DirectiveDefinition(directive.Name.Name); def != nil && def.FieldCollectionFilter != nil {
				if arguments, err := coerceArgumentValues(def.Arguments, directive.Arguments, e.VariableValues); err == nil && !def.FieldCollectionFilter(arguments) {
					skip = true
				}
			}
		}
		if skip {
			continue
		}

		switch selection := selection.(type) {
		case *ast.Field:
			responseKey := selection.Name.Name
			if selection.Alias != nil {
				responseKey = selection.Alias.Name
			}
			if groupForResponseKey, ok := groupedFields.Get(responseKey); ok {
				groupedFields.Set(responseKey, append(groupForResponseKey.([]*ast.Field), selection))
			} else {
				groupedFields.Set(responseKey, []*ast.Field{selection})
			}
		case *ast.FragmentSpread:
			fragmentSpreadName := selection.FragmentName.Name
			if _, ok := visitedFragments[fragmentSpreadName]; ok {
				continue
			}
			visitedFragments[fragmentSpreadName] = struct{}{}

			fragment := e.FragmentDefinitions[fragmentSpreadName]
			if fragment == nil {
				continue
			}

			fragmentType := schemaType(fragment.TypeCondition, e.Schema)
			if fragmentType == nil || !doesFragmentTypeApply(objectType, fragmentType) {
				continue
			}

			e.collectFields(objectType, fragment.SelectionSet.Selections, visitedFragments, groupedFields)
		case *ast.InlineFragment:
			if selection.TypeCondition != nil {
				fragmentType := schemaType(selection.TypeCondition, e.Schema)
				if fragmentType == nil || !doesFragmentTypeApply(objectType, fragmentType) {
					continue
				}
			}

			e.collectFields(objectType, selection.SelectionSet.Selections, visitedFragments, groupedFields)
		default:
			panic(fmt.Sprintf("unexpected selection type: %T", selection))
		}
	}
}

func doesFragmentTypeApply(objectType *schema.ObjectType, fragmentType schema.Type) bool {
	switch fragmentType := fragmentType.(type) {
	case *schema.ObjectType:
		return objectType.IsSameType(fragmentType)
	case *schema.InterfaceType:
		for _, impl := range objectType.ImplementedInterfaces {
			if impl.IsSameType(fragmentType) {
				return true
			}
		}
		return false
	case *schema.UnionType:
		for _, member := range fragmentType.MemberTypes {
			if member.IsSameType(objectType) {
				return true
			}
		}
		return false
	}
	panic(fmt.Sprintf("unexpected fragment type: %T", fragmentType))
}

func getOperation(doc *ast.Document, operationName string) (*ast.OperationDefinition, *Error) {
	var ret *ast.OperationDefinition
	for _, def := range doc.Definitions {
		if def, ok := def.(*ast.OperationDefinition); ok {
			if (def.Name == nil && operationName == "") || (def.Name != nil && def.Name.Name == operationName) {
				if ret != nil {
					return nil, newError("multiple matching operations")
				}
				ret = def
			}
		}
	}
	if ret == nil {
		return nil, newError("no matching operations")
	}
	return ret, nil
}

func schemaType(t ast.Type, s *schema.Schema) schema.Type {
	switch t := t.(type) {
	case *ast.ListType:
		if inner := schemaType(t.Type, s); inner != nil {
			return schema.NewListType(inner)
		}
	case *ast.NonNullType:
		if inner := schemaType(t.Type, s); inner != nil {
			return schema.NewNonNullType(inner)
		}
	case *ast.NamedType:
		return s.NamedType(t.Name.Name)
	default:
		panic(fmt.Sprintf("unexpected ast type: %T", t))
	}
	return nil
}

func coerceVariableValues(s *schema.Schema, operation *ast.OperationDefinition, variableValues map[string]interface{}) (map[string]interface{}, *Error) {
	coercedValues := map[string]interface{}{}
	for _, def := range operation.VariableDefinitions {
		variableName := def.Variable.Name.Name
		variableType := schemaType(def.Type, s)
		if variableType == nil || !variableType.IsInputType() {
			return nil, newError("invalid variable type")
		}
		value, hasValue := variableValues[variableName]

		if !hasValue && def.DefaultValue != nil {
			if coerced, err := schema.CoerceLiteral(def.DefaultValue, variableType, variableValues); err != nil {
				return nil, newError("%v", err.Error())
			} else {
				coercedValues[variableName] = coerced
			}
			continue
		} else if schema.IsNonNullType(variableType) && !hasValue {
			return nil, newError("the %v variable is required", variableName)
		} else if hasValue {
			if coerced, err := schema.CoerceVariableValue(value, variableType); err != nil {
				return nil, newError("%v", err.Error())
			} else {
				coercedValues[variableName] = coerced
			}
		}
	}
	return coercedValues, nil
}

func coerceArgumentValues(argumentDefinitions map[string]*schema.InputValueDefinition, arguments []*ast.Argument, variableValues map[string]interface{}) (map[string]interface{}, error) {
	coercedValues := map[string]interface{}{}

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
			coercedValues[argumentName] = defaultValue
		} else if schema.IsNonNullType(argumentType) && !hasValue {
			return nil, fmt.Errorf("the %v argument is required", argumentName)
		} else if hasValue {
			if argVariable, ok := argumentValue.(*ast.Variable); ok {
				coercedValues[argumentName] = variableValues[argVariable.Name.Name]
			} else if coerced, err := schema.CoerceLiteral(argumentValue, argumentType, variableValues); err != nil {
				return nil, err
			} else {
				coercedValues[argumentName] = coerced
			}
		}
	}

	return coercedValues, nil
}
