package graphql

import (
	"context"

	"github.com/ccbrown/api-fu/graphql/ast"
	"github.com/ccbrown/api-fu/graphql/executor"
	"github.com/ccbrown/api-fu/graphql/parser"
	"github.com/ccbrown/api-fu/graphql/schema"
	"github.com/ccbrown/api-fu/graphql/validator"
)

type Type = schema.Type
type ObjectType = schema.ObjectType
type InterfaceType = schema.InterfaceType
type EnumType = schema.EnumType
type ScalarType = schema.ScalarType
type UnionType = schema.UnionType
type InputObjectType = schema.InputObjectType
type NonNullType = schema.NonNullType
type ListType = schema.ListType

type FieldContext = schema.FieldContext
type InputValueDefinition = schema.InputValueDefinition
type FieldDefinition = schema.FieldDefinition

var IDType = schema.IDType

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

type Location struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

type Error struct {
	Message   string        `json:"message"`
	Locations []Location    `json:"locations,omitempty"`
	Path      []interface{} `json:"path,omitempty"`
}

type Response struct {
	Data   *interface{} `json:"data,omitempty"`
	Errors []*Error     `json:"error,omitempty"`
}

func Execute(r *Request) *Response {
	ret := &Response{}
	doc := r.Document
	if doc == nil {
		parsed, parseErrs := parser.ParseDocument([]byte(r.Query))
		if len(parseErrs) > 0 {
			for _, err := range parseErrs {
				ret.Errors = append(ret.Errors, &Error{
					Message: err.Message,
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
					Message:   err.Message,
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
		ret.Errors = append(ret.Errors, &Error{
			Message:   err.Message,
			Locations: locations,
			Path:      err.Path,
		})
	}
	return ret
}
