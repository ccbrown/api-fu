package validator

import (
	"fmt"

	"github.com/ccbrown/api-fu/graphql/ast"
	"github.com/ccbrown/api-fu/graphql/schema"
)

type Location struct {
	Line   int
	Column int
}

type Error struct {
	Message   string
	Locations []Location

	// If a validator is unable to perform its job due to an error unrelated to its purpose, it will
	// emit a secondary error. Secondary errors are always errors that should be caught by other
	// validators, so if there are any primary errors, secondary errors are discarded as they should
	// all be duplicates. If a secondary error makes it out of validation, there's probably a
	// mistake in one of the validators.
	isSecondary bool
}

func (err *Error) Error() string {
	return err.Message
}

func locationsForNodes(nodes ...ast.Node) []Location {
	if len(nodes) == 0 {
		return nil
	}
	ret := make([]Location, len(nodes))
	for i, node := range nodes {
		ret[i].Line = node.Position().Line
		ret[i].Column = node.Position().Column
	}
	return ret
}

func newError(node ast.Node, message string, args ...interface{}) *Error {
	return &Error{
		Message:   fmt.Sprintf(message, args...),
		Locations: locationsForNodes(node),
	}
}

func newErrorWithNodes(nodes []ast.Node, message string, args ...interface{}) *Error {
	return &Error{
		Message:   fmt.Sprintf(message, args...),
		Locations: locationsForNodes(nodes...),
	}
}

func newSecondaryError(node ast.Node, message string, args ...interface{}) *Error {
	return &Error{
		Message:     fmt.Sprintf(message, args...),
		isSecondary: true,
		Locations:   locationsForNodes(node),
	}
}

type Rule func(*ast.Document, *schema.Schema, schema.FeatureSet, *TypeInfo) []*Error

func ValidateDocument(doc *ast.Document, s *schema.Schema, features schema.FeatureSet, additionalRules ...Rule) []*Error {
	typeInfo := NewTypeInfo(doc, s, features)
	var errs []*Error
	for _, f := range append([]Rule{
		validateDocument,
		validateOperations,
		validateFields,
		validateArguments,
		validateFragments,
		validateValues,
		validateDirectives,
		validateVariables,
	}, additionalRules...) {
		errs = append(errs, f(doc, s, features, typeInfo)...)
	}
	var primary []*Error
	for _, err := range errs {
		if !err.isSecondary {
			primary = append(primary, err)
		}
	}
	if len(primary) > 0 {
		return primary
	}
	return errs
}
