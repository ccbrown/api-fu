package executor

import (
	"context"
	"fmt"
	"reflect"
	"strconv"

	"github.com/ccbrown/api-fu/graphql/ast"
	"github.com/ccbrown/api-fu/graphql/executor/internal/future"
	"github.com/ccbrown/api-fu/graphql/schema"
	"github.com/ccbrown/api-fu/graphql/schema/introspection"
)

// ResolveResult represents the result of a field resolver. This type is generally used with
// ResolvePromise to pass around asynchronous results.
type ResolveResult struct {
	Value interface{}
	Error error
}

// ResolvePromise can be used to resolve fields asynchronously. You may return ResolvePromise from
// the field's resolve function. If you do, you must define an IdleHandler for the request. Any time
// request execution is unable to proceed, the idle handler will be invoked. Before the idle handler
// returns, a result must be sent to at least one previously returned ResolvePromise.
type ResolvePromise chan ResolveResult

// Request defines all of the inputs required to execute a GraphQL query.
type Request struct {
	Document       *ast.Document
	Schema         *schema.Schema
	OperationName  string
	VariableValues map[string]interface{}
	InitialValue   interface{}
	IdleHandler    func()
}

// ExecuteRequest executes a request.
func ExecuteRequest(ctx context.Context, r *Request) (*OrderedMap, []*Error) {
	if e, err := newExecutor(ctx, r); err != nil {
		return nil, []*Error{err}
	} else if opType := e.Operation.OperationType; opType == nil || opType.Value == "query" {
		return e.executeQuery(r.InitialValue)
	} else if opType.Value == "mutation" {
		return e.executeMutation(r.InitialValue)
	} else if opType.Value == "subscription" {
		return e.executeSubscriptionEvent(r.InitialValue)
	}
	panic("unexpected operation type")
}

// IsSubscription can be used to determine if a request is for a subscription.
func IsSubscription(doc *ast.Document, operationName string) bool {
	operation, err := GetOperation(doc, operationName)
	return err == nil && operation.OperationType != nil && operation.OperationType.Value == "subscription"
}

// Subscribe resolves the root subscription field of a request and returns the result.
func Subscribe(ctx context.Context, r *Request) (interface{}, *Error) {
	if e, err := newExecutor(ctx, r); err != nil {
		return nil, err
	} else if e.Operation.OperationType != nil && e.Operation.OperationType.Value == "subscription" {
		return e.subscribe(r.InitialValue)
	} else {
		return nil, newError(e.Operation, "A subscription operation is required.")
	}
}

type executor struct {
	Context             context.Context
	Schema              *schema.Schema
	FragmentDefinitions map[string]*ast.FragmentDefinition
	VariableValues      map[string]interface{}
	Errors              []*Error
	Operation           *ast.OperationDefinition
	IdleHandler         func()

	// GroupedFieldSetCache is used to cache the results of collectFields.
	GroupedFieldSetCache map[string]*GroupedFieldSet

	// CatchError is used to handle errors for nullable fields. The closure is generated on
	// construction to avoid allocations during execution.
	CatchError func(future.Result) future.Result
}

func newExecutor(ctx context.Context, r *Request) (*executor, *Error) {
	operation, err := GetOperation(r.Document, r.OperationName)
	if err != nil {
		return nil, err
	}
	coercedVariableValues, err := coerceVariableValues(r.Schema, operation, r.VariableValues)
	if err != nil {
		return nil, err
	}

	e := &executor{
		Context:              ctx,
		Schema:               r.Schema,
		FragmentDefinitions:  map[string]*ast.FragmentDefinition{},
		VariableValues:       coercedVariableValues,
		Operation:            operation,
		IdleHandler:          r.IdleHandler,
		GroupedFieldSetCache: map[string]*GroupedFieldSet{},
	}
	e.CatchError = func(r future.Result) future.Result {
		if r.IsErr() {
			e.Errors = append(e.Errors, r.Error.(*Error))
			r.Error = nil
		}
		return r
	}
	for _, def := range r.Document.Definitions {
		if def, ok := def.(*ast.FragmentDefinition); ok {
			e.FragmentDefinitions[def.Name.Name] = def
		}
	}
	return e, nil
}

func (e *executor) executeQuery(initialValue interface{}) (*OrderedMap, []*Error) {
	queryType := e.Schema.QueryType()
	if !schema.IsObjectType(queryType) {
		return nil, []*Error{newError(e.Operation, "This schema cannot perform queries.")}
	}
	if data, err := e.wait(e.executeSelections(e.Operation.SelectionSet.Selections, queryType, initialValue, nil, false)); err != nil {
		e.Errors = append(e.Errors, err.(*Error))
		return nil, e.Errors
	} else if data != nil {
		return data.(*OrderedMap), e.Errors
	}
	return nil, nil
}

func (e *executor) executeMutation(initialValue interface{}) (*OrderedMap, []*Error) {
	mutationType := e.Schema.MutationType()
	if !schema.IsObjectType(mutationType) {
		return nil, []*Error{newError(e.Operation, "This schema cannot perform mutations.")}
	}
	if data, err := e.wait(e.executeSelections(e.Operation.SelectionSet.Selections, mutationType, initialValue, nil, true)); err != nil {
		e.Errors = append(e.Errors, err.(*Error))
		return nil, e.Errors
	} else if data != nil {
		return data.(*OrderedMap), e.Errors
	}
	return nil, nil
}

func (e *executor) subscribe(initialValue interface{}) (interface{}, *Error) {
	subscriptionType := e.Schema.SubscriptionType()
	if !schema.IsObjectType(subscriptionType) {
		return nil, newError(e.Operation, "This schema cannot perform subscriptions.")
	}

	groupedFieldSet := e.collectFields(subscriptionType, e.Operation.SelectionSet.Selections)

	if groupedFieldSet.Len() != 1 {
		return nil, newError(e.Operation.SelectionSet, "Subscriptions must contain exactly one root field selection.")
	}

	item := groupedFieldSet.Items()[0]
	fields := item.Fields
	field := fields[0]
	fieldName := field.Name.Name
	fieldDef := subscriptionType.Fields[fieldName]
	if fieldDef == nil {
		return nil, newError(field, "Undefined root subscription field.")
	}
	argumentValues, err := coerceArgumentValues(field, fieldDef.Arguments, field.Arguments, e.VariableValues)
	if err != nil {
		return nil, err
	}

	resolveValue, resolveErr := fieldDef.Resolve(&schema.FieldContext{
		Context:     e.Context,
		Schema:      e.Schema,
		Object:      initialValue,
		Arguments:   argumentValues,
		IsSubscribe: true,
	})
	if !isNil(resolveErr) {
		return nil, &Error{
			Message: err.Error(),
			Locations: []Location{{
				Line:   field.Position().Line,
				Column: field.Position().Column,
			}},
			Path:          []interface{}{item.Key},
			originalError: resolveErr,
		}
	}
	return resolveValue, nil
}

func (e *executor) executeSubscriptionEvent(initialValue interface{}) (*OrderedMap, []*Error) {
	subscriptionType := e.Schema.SubscriptionType()
	if !schema.IsObjectType(subscriptionType) {
		return nil, []*Error{newError(e.Operation, "This schema cannot perform subscriptions.")}
	}
	if data, err := e.wait(e.executeSelections(e.Operation.SelectionSet.Selections, subscriptionType, initialValue, nil, false)); err != nil {
		e.Errors = append(e.Errors, err.(*Error))
		return nil, e.Errors
	} else if data != nil {
		return data.(*OrderedMap), e.Errors
	}
	return nil, nil
}

func (e *executor) wait(f future.Future) (interface{}, error) {
	var result future.Result
	done := false
	f = f.Map(func(r future.Result) future.Result {
		result = r
		done = true
		return r
	})
	f.Poll()
	for !done {
		if e.IdleHandler == nil {
			return nil, newError(nil, "No idle handler defined.")
		}
		e.IdleHandler()
		f.Poll()
	}
	return result.Value, result.Error
}

// executeSelections returns a future for an *OrderedMap
func (e *executor) executeSelections(selections []ast.Selection, objectType *schema.ObjectType, objectValue interface{}, path *path, forceSerial bool) future.Future {
	groupedFieldSet := e.collectFields(objectType, selections)

	resultMap := NewOrderedMapWithLength(groupedFieldSet.Len())

	futures := make([]future.Future, 0, groupedFieldSet.Len())

	for i, item := range groupedFieldSet.Items() {
		responseKey := item.Key
		fields := item.Fields
		fieldName := fields[0].Name.Name

		if fieldName == "__typename" {
			resultMap.Set(i, responseKey, objectType.Name)
			continue
		}

		fieldDef := objectType.Fields[fieldName]
		if fieldDef == nil && objectType == e.Schema.QueryType() {
			fieldDef = introspection.MetaFields[fieldName]
		}

		if fieldDef != nil {
			f := e.catchErrorIfNullable(fieldDef.Type, e.executeField(objectValue, fields, fieldDef, path.WithStringComponent(responseKey)))
			if forceSerial {
				responseValue, err := e.wait(f)
				if err != nil {
					return future.Err(err)
				}
				resultMap.Set(i, responseKey, responseValue)
			} else {
				i := i
				responseKey := responseKey
				futures = append(futures, f.MapOk(func(responseValue interface{}) interface{} {
					resultMap.Set(i, responseKey, responseValue)
					return nil
				}))
			}
		}
	}

	return future.After(futures...).MapOk(func(interface{}) interface{} {
		return resultMap
	})
}

func isNil(v interface{}) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	return (rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface) && rv.IsNil()
}

func newFieldResolveError(fields []*ast.Field, err error, path *path) *Error {
	locations := make([]Location, len(fields))
	for i, field := range fields {
		locations[i].Line = field.Position().Line
		locations[i].Column = field.Position().Column
	}
	return &Error{
		Message:       err.Error(),
		Locations:     locations,
		Path:          path.Slice(),
		originalError: err,
	}
}

func (e *executor) executeField(objectValue interface{}, fields []*ast.Field, fieldDef *schema.FieldDefinition, path *path) future.Future {
	field := fields[0]
	argumentValues, coercionErr := coerceArgumentValues(field, fieldDef.Arguments, field.Arguments, e.VariableValues)
	if coercionErr != nil {
		return future.Err(coercionErr)
	}
	resolvedValue, err := fieldDef.Resolve(&schema.FieldContext{
		Context:   e.Context,
		Schema:    e.Schema,
		Object:    objectValue,
		Arguments: argumentValues,
	})
	if !isNil(err) {
		return future.Err(newFieldResolveError(fields, err, path))
	}
	if f, ok := resolvedValue.(ResolvePromise); ok {
		return future.New(func() (future.Result, bool) {
			var result future.Result
			select {
			case r := <-f:
				if !isNil(r.Error) {
					result.Error = r.Error
				} else {
					result.Value = r.Value
				}
				return result, true
			default:
				return result, false
			}
		}).Then(func(r future.Result) future.Future {
			if r.IsOk() {
				return e.completeValue(fieldDef.Type, fields, r.Value, path)
			}
			return future.Err(newFieldResolveError(fields, r.Error, path))
		})
	}
	return e.completeValue(fieldDef.Type, fields, resolvedValue, path)
}

func (e *executor) catchErrorIfNullable(t schema.Type, f future.Future) future.Future {
	if schema.IsNonNullType(t) {
		return f
	}
	return f.Map(e.CatchError)
}

func (e *executor) completeValue(fieldType schema.Type, fields []*ast.Field, result interface{}, path *path) future.Future {
	if nonNullType, ok := fieldType.(*schema.NonNullType); ok {
		return e.completeValue(nonNullType.Type, fields, result, path).Map(func(r future.Result) future.Result {
			if r.IsOk() && r.Value == nil {
				r.Error = newErrorWithPath(fields[0], path, "Null result for non-null field.")
			}
			return r
		})
	}

	if isNil(result) {
		return future.Ok(nil)
	}

	switch fieldType := fieldType.(type) {
	case *schema.ListType:
		result := reflect.ValueOf(result)
		if result.Kind() != reflect.Slice {
			return future.Err(newErrorWithPath(fields[0], path, "Result is not a list."))
		}
		innerType := fieldType.Type
		completedResult := make([]future.Future, result.Len())
		for i := range completedResult {
			completedResult[i] = e.catchErrorIfNullable(innerType, e.completeValue(innerType, fields, result.Index(i).Interface(), path.WithIntComponent(i)))
		}
		return future.Join(completedResult...)
	case *schema.ScalarType:
		coerced, err := fieldType.CoerceResult(result)
		if err != nil {
			return future.Err(newErrorWithPath(fields[0], path, "Unexpected result: %v", err))
		}
		return future.Ok(coerced)
	case *schema.EnumType:
		coerced, err := fieldType.CoerceResult(result)
		if err != nil {
			return future.Err(newErrorWithPath(fields[0], path, "Unexpected result: %v", err))
		}
		return future.Ok(coerced)
	case *schema.ObjectType, *schema.InterfaceType, *schema.UnionType:
		var objectType *schema.ObjectType
		switch fieldType := fieldType.(type) {
		case *schema.ObjectType:
			objectType = fieldType
		case *schema.InterfaceType:
			for _, t := range e.Schema.InterfaceImplementations(fieldType.Name) {
				if t.IsTypeOf(result) {
					objectType = t
					break
				}
			}
		case *schema.UnionType:
			for _, t := range fieldType.MemberTypes {
				if t.IsTypeOf(result) {
					objectType = t
					break
				}
			}
		}
		if objectType == nil {
			return future.Err(newErrorWithPath(fields[0], path, "Unable to determine object type."))
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

func (e *executor) collectFields(objectType *schema.ObjectType, selections []ast.Selection) *GroupedFieldSet {
	// collectFields can be called many times with the same inputs throughout a query's execution,
	// so we memoize the return value.

	cacheKey := objectType.Name
	for _, sel := range selections {
		pos := sel.Position()
		cacheKey += ":" + strconv.Itoa(pos.Line) + "," + strconv.Itoa(pos.Column)
	}

	if hit, ok := e.GroupedFieldSetCache[cacheKey]; ok {
		return hit
	}

	groupedFieldSet := NewGroupedFieldSetWithCapacity(len(selections))
	e.collectFieldsImpl(objectType, selections, nil, groupedFieldSet)
	e.GroupedFieldSetCache[cacheKey] = groupedFieldSet
	return groupedFieldSet
}

func (e *executor) collectFieldsImpl(objectType *schema.ObjectType, selections []ast.Selection, visitedFragments map[string]struct{}, groupedFields *GroupedFieldSet) {
	if visitedFragments == nil {
		visitedFragments = map[string]struct{}{}
	}
	for _, selection := range selections {
		skip := false
		for _, directive := range selection.SelectionDirectives() {
			if def := e.Schema.Directives()[directive.Name.Name]; def != nil && def.FieldCollectionFilter != nil {
				if arguments, err := coerceArgumentValues(directive, def.Arguments, directive.Arguments, e.VariableValues); err == nil && !def.FieldCollectionFilter(arguments) {
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
			groupedFields.Append(responseKey, selection)
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

			e.collectFieldsImpl(objectType, fragment.SelectionSet.Selections, visitedFragments, groupedFields)
		case *ast.InlineFragment:
			if selection.TypeCondition != nil {
				fragmentType := schemaType(selection.TypeCondition, e.Schema)
				if fragmentType == nil || !doesFragmentTypeApply(objectType, fragmentType) {
					continue
				}
			}

			e.collectFieldsImpl(objectType, selection.SelectionSet.Selections, visitedFragments, groupedFields)
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

// GetOperation returns the operation selected by the given name. If operationName is "" and the
// document contains only one operation, it is returned. Otherwise the document must contain exactly
// one operation with the given name.
func GetOperation(doc *ast.Document, operationName string) (*ast.OperationDefinition, *Error) {
	var ret *ast.OperationDefinition
	for _, def := range doc.Definitions {
		if def, ok := def.(*ast.OperationDefinition); ok {
			if operationName == "" || (def.Name != nil && def.Name.Name == operationName) {
				if ret != nil {
					return nil, newError(def, "Multiple matching operations.")
				}
				ret = def
			}
		}
	}
	if ret == nil {
		return nil, newError(nil, "No matching operations.")
	}
	return ret, nil
}

func namedType(s *schema.Schema, name string) schema.NamedType {
	if ret := s.NamedTypes()[name]; ret != nil {
		return ret
	}
	return introspection.NamedTypes[name]
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
		return namedType(s, t.Name.Name)
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

func coerceArgumentValues(node ast.Node, argumentDefinitions map[string]*schema.InputValueDefinition, arguments []*ast.Argument, variableValues map[string]interface{}) (map[string]interface{}, *Error) {
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
