package executor

import (
	"context"
	"encoding/binary"
	"fmt"
	"reflect"

	"github.com/ccbrown/api-fu/graphql/ast"
	"github.com/ccbrown/api-fu/graphql/executor/internal/future"
	"github.com/ccbrown/api-fu/graphql/schema"
	"github.com/ccbrown/api-fu/graphql/schema/introspection"
	"github.com/ccbrown/api-fu/graphql/validator"
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
	CatchError func(future.Result[any]) future.Result[any]
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
	e.CatchError = func(r future.Result[any]) future.Result[any] {
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
	if data, err := wait(e, e.executeSelections(e.Operation.SelectionSet.Selections, queryType, initialValue, nil, false)); err != nil {
		e.Errors = append(e.Errors, err.(*Error))
		return nil, e.Errors
	} else if data != nil {
		return data, e.Errors
	}
	return nil, nil
}

func (e *executor) executeMutation(initialValue interface{}) (*OrderedMap, []*Error) {
	mutationType := e.Schema.MutationType()
	if !schema.IsObjectType(mutationType) {
		return nil, []*Error{newError(e.Operation, "This schema cannot perform mutations.")}
	}
	if data, err := wait(e, e.executeSelections(e.Operation.SelectionSet.Selections, mutationType, initialValue, nil, true)); err != nil {
		e.Errors = append(e.Errors, err.(*Error))
		return nil, e.Errors
	} else if data != nil {
		return data, e.Errors
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
			Message: resolveErr.Error(),
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
	if data, err := wait(e, e.executeSelections(e.Operation.SelectionSet.Selections, subscriptionType, initialValue, nil, false)); err != nil {
		e.Errors = append(e.Errors, err.(*Error))
		return nil, e.Errors
	} else if data != nil {
		return data, e.Errors
	}
	return nil, nil
}

func wait[T any](e *executor, f future.Future[T]) (T, error) {
	var result future.Result[T]
	done := false
	f = future.Map(f, func(r future.Result[T]) future.Result[T] {
		result = r
		done = true
		return r
	})
	f.Poll()
	for !done {
		if e.IdleHandler == nil {
			return result.Value, newError(nil, "No idle handler defined.")
		}
		e.IdleHandler()
		f.Poll()
	}
	return result.Value, result.Error
}

func (e *executor) executeSelections(selections []ast.Selection, objectType *schema.ObjectType, objectValue interface{}, path *path, forceSerial bool) future.Future[*OrderedMap] {
	groupedFieldSet := e.collectFields(objectType, selections)

	resultMap := NewOrderedMapWithLength(groupedFieldSet.Len())

	futures := make([]future.Future[any], 0, groupedFieldSet.Len())

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
				responseValue, err := wait(e, f)
				if err != nil {
					return future.Err[*OrderedMap](err)
				}
				resultMap.Set(i, responseKey, responseValue)
			} else {
				i := i
				responseKey := responseKey
				futures = append(futures, future.MapOk(f, func(responseValue any) any {
					resultMap.Set(i, responseKey, responseValue)
					return nil
				}))
			}
		}
	}

	return future.MapOk(future.After(futures...), func(struct{}) *OrderedMap {
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

func (e *executor) executeField(objectValue interface{}, fields []*ast.Field, fieldDef *schema.FieldDefinition, path *path) future.Future[any] {
	field := fields[0]
	argumentValues, coercionErr := coerceArgumentValues(field, fieldDef.Arguments, field.Arguments, e.VariableValues)
	if coercionErr != nil {
		return future.Err[any](coercionErr)
	}
	if err := e.Context.Err(); err != nil {
		return future.Err[any](newFieldResolveError(fields, err, path))
	}
	resolvedValue, err := fieldDef.Resolve(&schema.FieldContext{
		Context:   e.Context,
		Schema:    e.Schema,
		Object:    objectValue,
		Arguments: argumentValues,
	})
	if !isNil(err) {
		return future.Err[any](newFieldResolveError(fields, err, path))
	}
	if f, ok := resolvedValue.(ResolvePromise); ok {
		return future.Then(future.New(func() (future.Result[any], bool) {
			var result future.Result[any]
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
		}), func(r future.Result[any]) future.Future[any] {
			if r.IsOk() {
				return e.completeValue(fieldDef.Type, fields, r.Value, path)
			}
			return future.Err[any](newFieldResolveError(fields, r.Error, path))
		})
	}
	return e.completeValue(fieldDef.Type, fields, resolvedValue, path)
}

func (e *executor) catchErrorIfNullable(t schema.Type, f future.Future[any]) future.Future[any] {
	if schema.IsNonNullType(t) {
		return f
	}
	return future.Map(f, e.CatchError)
}

func (e *executor) completeValue(fieldType schema.Type, fields []*ast.Field, result interface{}, path *path) future.Future[any] {
	if nonNullType, ok := fieldType.(*schema.NonNullType); ok {
		return future.Map(e.completeValue(nonNullType.Type, fields, result, path), func(r future.Result[any]) future.Result[any] {
			if r.IsOk() && r.Value == nil {
				r.Error = newErrorWithPath(fields[0], path, "Null result for non-null field.")
			}
			return r
		})
	}

	if isNil(result) {
		return future.Ok[any](nil)
	}

	switch fieldType := fieldType.(type) {
	case *schema.ListType:
		result := reflect.ValueOf(result)
		if result.Kind() != reflect.Slice {
			return future.Err[any](newErrorWithPath(fields[0], path, "Result is not a list."))
		}
		innerType := fieldType.Type
		completedResult := make([]future.Future[any], result.Len())
		for i := range completedResult {
			completedResult[i] = e.catchErrorIfNullable(innerType, e.completeValue(innerType, fields, result.Index(i).Interface(), path.WithIntComponent(i)))
		}
		return future.MapOk(future.Join(completedResult...), func(l []interface{}) interface{} {
			return l
		})
	case *schema.ScalarType:
		coerced, err := fieldType.CoerceResult(result)
		if err != nil {
			return future.Err[any](newErrorWithPath(fields[0], path, "Unexpected result: %v", err))
		}
		return future.Ok(coerced)
	case *schema.EnumType:
		coerced, err := fieldType.CoerceResult(result)
		if err != nil {
			return future.Err[any](newErrorWithPath(fields[0], path, "Unexpected result: %v", err))
		}
		return future.Ok[any](coerced)
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
			return future.Err[any](newErrorWithPath(fields[0], path, "Unable to determine object type."))
		}
		return future.MapOk(e.executeSelections(mergeSelectionSets(fields), objectType, result, path, false), func(m *OrderedMap) interface{} {
			return m
		})
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

	cacheKeyBytes := make([]byte, len(objectType.Name)+16*len(selections))
	copy(cacheKeyBytes, objectType.Name)
	for i, sel := range selections {
		pos := sel.Position()
		binary.LittleEndian.PutUint64(cacheKeyBytes[len(objectType.Name)+i*16:], uint64(pos.Line))
		binary.LittleEndian.PutUint64(cacheKeyBytes[len(objectType.Name)+i*16+8:], uint64(pos.Column))
	}
	cacheKey := string(cacheKeyBytes)

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
	ret, err := validator.CoerceVariableValues(s, operation, variableValues)
	return ret, newErrorWithValidatorError(err)
}

func coerceArgumentValues(node ast.Node, argumentDefinitions map[string]*schema.InputValueDefinition, arguments []*ast.Argument, variableValues map[string]interface{}) (map[string]interface{}, *Error) {
	ret, err := validator.CoerceArgumentValues(node, argumentDefinitions, arguments, variableValues)
	return ret, newErrorWithValidatorError(err)
}
