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

type Type = schema.Type
type NamedType = schema.NamedType
type ObjectType = schema.ObjectType
type InterfaceType = schema.InterfaceType
type EnumType = schema.EnumType
type ScalarType = schema.ScalarType
type UnionType = schema.UnionType
type InputObjectType = schema.InputObjectType
type NonNullType = schema.NonNullType
type ListType = schema.ListType

type FieldContext = schema.FieldContext
type EnumValueDefinition = schema.EnumValueDefinition
type InputValueDefinition = schema.InputValueDefinition
type FieldDefinition = schema.FieldDefinition
type DirectiveDefinition = schema.DirectiveDefinition

var IncludeDirective = schema.IncludeDirective
var SkipDirective = schema.SkipDirective

var IDType = schema.IDType
var StringType = schema.StringType
var IntType = schema.IntType
var FloatType = schema.FloatType
var BooleanType = schema.BooleanType

func NewNonNullType(t Type) *NonNullType {
	return schema.NewNonNullType(t)
}

func NewListType(t Type) *ListType {
	return schema.NewListType(t)
}

type Schema = schema.Schema
type SchemaDefinition = schema.SchemaDefinition

func NewSchema(def *SchemaDefinition) (*Schema, error) {
	return schema.New(def)
}

type Request struct {
	Context context.Context

	Query string

	// In some cases, you may want to optimize by providing the parsed and validated AST document
	// instead of Query.
	Document *ast.Document

	Schema         *Schema
	OperationName  string
	VariableValues map[string]interface{}
	InitialValue   interface{}
	IdleHandler    func()
}

func NewRequestFromHTTP(r *http.Request) (req *Request, code int, err error) {
	req = &Request{
		Context: r.Context(),
	}

	switch r.Method {
	case http.MethodGet:
		if query := r.URL.Query().Get("query"); query == "" {
			return nil, http.StatusBadRequest, fmt.Errorf("the query parameter is required")
		} else {
			req.Query = query
		}

		if variables := r.URL.Query().Get("variables"); variables != "" {
			if err := json.Unmarshal([]byte(variables), &req.VariableValues); err != nil {
				return nil, http.StatusBadRequest, fmt.Errorf("malformed variables parameter")
			}
		}

		req.OperationName = r.URL.Query().Get("variables")
	case http.MethodPost:
		if query := r.URL.Query().Get("query"); query != "" {
			req.Query = query
		}

		switch mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); mediaType {
		case "application/json":
			var body struct {
				Query         string                 `json:"query"`
				OperationName string                 `json:"operationName"`
				Variables     map[string]interface{} `json:"variables"`
			}

			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				return nil, http.StatusBadRequest, fmt.Errorf("malformed request body")
			}

			req.Query = body.Query
			req.OperationName = body.OperationName
			req.VariableValues = body.Variables
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

type Location struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

type Error struct {
	Message   string        `json:"message"`
	Locations []Location    `json:"locations,omitempty"`
	Path      []interface{} `json:"path,omitempty"`

	// To populate this field, your resolvers can return errors that implement ExtendedError.
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// If a resolver returns an error that implements this interface, the error's extensions property
// will be populated.
type ExtendedError interface {
	error
	Extensions() map[string]interface{}
}

type Response struct {
	Data   *interface{} `json:"data,omitempty"`
	Errors []*Error     `json:"errors,omitempty"`
}

func Execute(r *Request) *Response {
	ret := &Response{}
	doc := r.Document
	if doc == nil {
		parsed, parseErrs := parser.ParseDocument([]byte(r.Query))
		if len(parseErrs) > 0 {
			for _, err := range parseErrs {
				ret.Errors = append(ret.Errors, &Error{
					Message: "Syntax error: " + err.Message,
					Locations: []Location{
						Location{
							Line:   err.Location.Line,
							Column: err.Location.Column,
						},
					},
				})
			}
			return ret
		}
		if validationErrs := validator.ValidateDocument(parsed, r.Schema); len(validationErrs) > 0 {
			for _, err := range validationErrs {
				locations := make([]Location, len(err.Locations))
				for i, loc := range err.Locations {
					locations[i].Line = loc.Line
					locations[i].Column = loc.Column
				}
				ret.Errors = append(ret.Errors, &Error{
					Message:   "Validation error: " + err.Message,
					Locations: locations,
				})
			}
			return ret
		}
		doc = parsed
	}

	data, errs := executor.ExecuteRequest(r.Context, &executor.Request{
		Document:       doc,
		Schema:         r.Schema,
		OperationName:  r.OperationName,
		VariableValues: r.VariableValues,
		InitialValue:   r.InitialValue,
		IdleHandler:    r.IdleHandler,
	})
	var dataInterface interface{}
	dataInterface = data
	ret.Data = &dataInterface
	for _, err := range errs {
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
		ret.Errors = append(ret.Errors, retErr)
	}
	return ret
}
