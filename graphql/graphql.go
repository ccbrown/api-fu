package graphql

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"

	"github.com/ccbrown/api-fu/graphql/ast"
	"github.com/ccbrown/api-fu/graphql/executor"
	"github.com/ccbrown/api-fu/graphql/parser"
	"github.com/ccbrown/api-fu/graphql/schema"
	"github.com/ccbrown/api-fu/graphql/validator"
)

// Directive represents a GraphQL directive.
type Directive = schema.Directive

// Type represents a GraphQL type.
type Type = schema.Type

// NamedType represents any GraphQL named type.
type NamedType = schema.NamedType

// ObjectType represents a GraphQL object type.
type ObjectType = schema.ObjectType

// InterfaceType represents a GraphQL interface type.
type InterfaceType = schema.InterfaceType

// EnumType represents a GraphQL enum type.
type EnumType = schema.EnumType

// ScalarType represents a GraphQL scalar type.
type ScalarType = schema.ScalarType

// UnionType represents a GraphQL union type.
type UnionType = schema.UnionType

// InputObjectType represents a GraphQL input object type.
type InputObjectType = schema.InputObjectType

// NonNullType represents a non-null GraphQL type.
type NonNullType = schema.NonNullType

// ListType represents a GraphQL list type.
type ListType = schema.ListType

// FieldContext is provided to field resolvers and contains important context such as the current
// object and arguments.
type FieldContext = schema.FieldContext

// FieldCostContext contains important context passed to field cost functions.
type FieldCostContext = schema.FieldCostContext

// FieldCost describes the cost of resolving a field, enabling rate limiting and metering.
type FieldCost = schema.FieldCost

// Returns a cost function which returns a constant resolver cost with no multiplier.
func FieldResolverCost(n int) func(*FieldCostContext) FieldCost {
	return schema.FieldResolverCost(n)
}

// EnumValueDefinition defines a possible value for an enum type.
type EnumValueDefinition = schema.EnumValueDefinition

// InputValueDefinition defines an input value such as an argument.
type InputValueDefinition = schema.InputValueDefinition

// FieldDefinition defines a field on an object type.
type FieldDefinition = schema.FieldDefinition

// DirectiveDefinition defines a directive.
type DirectiveDefinition = schema.DirectiveDefinition

// ValidatorRule defines a rule that the validator will evaluate.
type ValidatorRule = validator.Rule

// Calculates the cost of the given operation and ensures it is not greater than max. If max is -1,
// no limit is enforced. If actual is non-nil, it is set to the actual cost of the operation.
// Queries with costs that are too high to calculate due to overflows always result in an error when
// max is non-negative, and actual will be set to the maximum possible value.
func ValidateCost(operationName string, variableValues map[string]interface{}, max int, actual *int) ValidatorRule {
	return validator.ValidateCost(operationName, variableValues, max, actual)
}

// IncludeDirective implements the @include directive as defined by the GraphQL spec.
var IncludeDirective = schema.IncludeDirective

// SkipDirective implements the @skip directive as defined by the GraphQL spec.
var SkipDirective = schema.SkipDirective

// IDType implements the ID type as defined by the GraphQL spec. It can be deserialized from a
// string or an integer type, but always serializes to a string.
var IDType = schema.IDType

// StringType implements the String type as defined by the GraphQL spec.
var StringType = schema.StringType

// IntType implements the Int type as defined by the GraphQL spec.
var IntType = schema.IntType

// FloatType implements the Float type as defined by the GraphQL spec.
var FloatType = schema.FloatType

// BooleanType implements the Boolean type as defined by the GraphQL spec.
var BooleanType = schema.BooleanType

// NewNonNullType creates a new non-null type with the given wrapped type.
func NewNonNullType(t Type) *NonNullType {
	return schema.NewNonNullType(t)
}

// NewListType creates a new list type with the given element type.
func NewListType(t Type) *ListType {
	return schema.NewListType(t)
}

// ResolveResult represents the result of a field resolver. This type is generally used with
// ResolvePromise to pass around asynchronous results.
type ResolveResult = executor.ResolveResult

// ResolvePromise can be used to resolve fields asynchronously. You may return ResolvePromise from
// the field's resolve function. If you do, you must define an IdleHandler for the request. Any time
// request execution is unable to proceed, the idle handler will be invoked. Before the idle handler
// returns, a result must be sent to at least one previously returned ResolvePromise.
type ResolvePromise = executor.ResolvePromise

// Schema represents a GraphQL schema.
type Schema = schema.Schema

// SchemaDefinition defines a GraphQL schema.
type SchemaDefinition = schema.SchemaDefinition

// NewSchema validates a schema definition and builds a Schema from it.
func NewSchema(def *SchemaDefinition) (*Schema, error) {
	return schema.New(def)
}

// Request defines all of the inputs required to execute a GraphQL query.
type Request struct {
	Context context.Context

	Query string

	// In some cases, you may want to optimize by providing the parsed and validated AST document
	// instead of Query.
	Document *ast.Document

	Schema         *Schema
	OperationName  string
	VariableValues map[string]interface{}
	Extensions     map[string]interface{}
	InitialValue   interface{}
	IdleHandler    func()
}

// Calculates the cost of the requested operation and ensures it is not greater than max. If max is
// -1, no limit is enforced. If actual is non-nil, it is set to the actual cost of the operation.
// Queries with costs that are too high to calculate due to overflows always result in an error when
// max is non-negative, and actual will be set to the maximum possible value.
func (r *Request) ValidateCost(max int, actual *int) ValidatorRule {
	return validator.ValidateCost(r.OperationName, r.VariableValues, max, actual)
}

func (r *Request) executorRequest(doc *ast.Document) *executor.Request {
	return &executor.Request{
		Document:       doc,
		Schema:         r.Schema,
		OperationName:  r.OperationName,
		VariableValues: r.VariableValues,
		InitialValue:   r.InitialValue,
		IdleHandler:    r.IdleHandler,
	}
}

// NewRequestFromHTTP constructs a Request from an HTTP request. Requests may be GET requests using
// query string parameters or POST requests with either the application/json or application/graphql
// content type. If the request is malformed, an HTTP error code and error are returned.
func NewRequestFromHTTP(r *http.Request) (req *Request, code int, err error) {
	req = &Request{
		Context: r.Context(),
	}

	switch r.Method {
	case http.MethodGet:
		req.Query = r.URL.Query().Get("query")

		if variables := r.URL.Query().Get("variables"); variables != "" {
			if err := json.Unmarshal([]byte(variables), &req.VariableValues); err != nil {
				return nil, http.StatusBadRequest, fmt.Errorf("malformed variables parameter")
			}
		}

		req.OperationName = r.URL.Query().Get("operationName")

		if extensions := r.URL.Query().Get("extensions"); extensions != "" {
			if err := json.Unmarshal([]byte(extensions), &req.Extensions); err != nil {
				return nil, http.StatusBadRequest, fmt.Errorf("malformed extensions parameter")
			}
		}
	case http.MethodPost:
		req.Query = r.URL.Query().Get("query")

		switch mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); mediaType {
		case "application/json":
			var body struct {
				Query         string                 `json:"query"`
				OperationName string                 `json:"operationName"`
				Variables     map[string]interface{} `json:"variables"`
				Extensions    map[string]interface{} `json:"extensions"`
			}

			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				return nil, http.StatusBadRequest, fmt.Errorf("malformed request body")
			}

			req.Query = body.Query
			req.OperationName = body.OperationName
			req.VariableValues = body.Variables
			req.Extensions = body.Extensions
		case "application/graphql":
			body, _ := ioutil.ReadAll(r.Body)
			req.Query = string(body)
		default:
			return nil, http.StatusBadRequest, fmt.Errorf("invalid content-type")
		}
	default:
		return nil, http.StatusMethodNotAllowed, fmt.Errorf("method not allowed")
	}

	return req, http.StatusOK, nil
}

// Location represents the location of a character within a query's source text.
type Location struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// Error represents a GraphQL error as defined by the spec.
type Error struct {
	Message   string        `json:"message"`
	Locations []Location    `json:"locations,omitempty"`
	Path      []interface{} `json:"path,omitempty"`

	// To populate this field, your resolvers can return errors that implement ExtendedError.
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

func (err *Error) Error() string {
	return err.Message
}

// ExtendedError can be used to add data to a GraphQL error. If a resolver returns an error that
// implements this interface, the error's extensions property will be populated.
type ExtendedError interface {
	error
	Extensions() map[string]interface{}
}

// Response represents the result of executing a GraphQL query.
type Response struct {
	Data   *interface{} `json:"data,omitempty"`
	Errors []*Error     `json:"errors,omitempty"`
}

// IsSubscription returns true if the operation with the given name is a subscription operation.
// operationName can be "", in which case true will be returned if the only operation in the
// document is a subscription. In any error case (such as multiple matching subscriptions), false is
// returned.
func IsSubscription(doc *ast.Document, operationName string) bool {
	return executor.IsSubscription(doc, operationName)
}

// ParseAndValidate parses and validates a query.
func ParseAndValidate(query string, schema *Schema, additionalRules ...ValidatorRule) (*ast.Document, []*Error) {
	var errors []*Error
	parsed, parseErrs := parser.ParseDocument([]byte(query))
	if len(parseErrs) > 0 {
		for _, err := range parseErrs {
			errors = append(errors, &Error{
				Message: "Syntax error: " + err.Message,
				Locations: []Location{
					{
						Line:   err.Location.Line,
						Column: err.Location.Column,
					},
				},
			})
		}
		return nil, errors
	}
	if validationErrs := validator.ValidateDocument(parsed, schema, additionalRules...); len(validationErrs) > 0 {
		for _, err := range validationErrs {
			locations := make([]Location, len(err.Locations))
			for i, loc := range err.Locations {
				locations[i].Line = loc.Line
				locations[i].Column = loc.Column
			}
			errors = append(errors, &Error{
				Message:   "Validation error: " + err.Message,
				Locations: locations,
			})
		}
		return nil, errors
	}
	return parsed, nil
}

func newErrorFromExecutorError(err *executor.Error) *Error {
	locations := make([]Location, len(err.Locations))
	for i, loc := range err.Locations {
		locations[i].Line = loc.Line
		locations[i].Column = loc.Column
	}
	retErr := &Error{
		Message:   err.Message,
		Locations: locations,
		Path:      err.Path,
	}
	if ext, ok := err.Unwrap().(ExtendedError); ok {
		retErr.Extensions = ext.Extensions()
	}
	return retErr
}

// Subscribe is used to implement subscription support. For subscribe operations (as indicated via
// IsSubscription), this should be invoked on Execute. On success it will return the result of
// executing the subscription field's resolver.
func Subscribe(r *Request) (interface{}, []*Error) {
	doc := r.Document
	if doc == nil {
		var errors []*Error
		doc, errors = ParseAndValidate(r.Query, r.Schema)
		if len(errors) > 0 {
			return nil, errors
		}
	}

	ret, err := executor.Subscribe(r.Context, r.executorRequest(doc))
	if err != nil {
		return nil, []*Error{newErrorFromExecutorError(err)}
	}
	return ret, nil
}

// Execute executes a query. If the request does not have a Document defined, the Query field will
// be parsed and validated.
func Execute(r *Request) *Response {
	ret := &Response{}
	doc := r.Document
	if doc == nil {
		var errors []*Error
		doc, errors = ParseAndValidate(r.Query, r.Schema)
		if len(errors) > 0 {
			return &Response{
				Errors: errors,
			}
		}
	}

	data, errs := executor.ExecuteRequest(r.Context, r.executorRequest(doc))
	var dataInterface interface{}
	dataInterface = data
	ret.Data = &dataInterface
	for _, err := range errs {
		ret.Errors = append(ret.Errors, newErrorFromExecutorError(err))
	}
	return ret
}
