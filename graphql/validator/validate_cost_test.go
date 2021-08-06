package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ccbrown/api-fu/graphql/parser"
	"github.com/ccbrown/api-fu/graphql/schema"
)

func TestValidateCost(t *testing.T) {
	s, err := schema.New(&schema.SchemaDefinition{
		Query: objectType,
		Directives: map[string]*schema.DirectiveDefinition{
			"include": schema.IncludeDirective,
			"skip":    schema.SkipDirective,
		},
	})
	require.NoError(t, err)

	for name, tc := range map[string]struct {
		Source         string
		OperationName  string
		VariableValues map[string]interface{}
		MaxCost        int
		ExpectedCost   int
		ExpectedErrors int
	}{
		"Simple": {
			Source: `{freeBoolean}`,
		},
		"TypeName": {
			Source: `{__typename t:__typename}`,
		},
		"Multiplier": {
			Source:       `{objects(first: 10) { int }}`,
			ExpectedCost: 1 + 10,
			MaxCost:      100,
		},
		"MultiplierNoSubselections": {
			Source:       `{objects(first: 10) { freeBoolean }}`,
			ExpectedCost: 1,
			MaxCost:      100,
		},
		"MultiplierNesting": {
			Source:       `{objects(first: 10) { int objects(first: 5) { int } }}`,
			ExpectedCost: 1 + 10*(2+5),
			MaxCost:      100,
		},
		"MaxExceeded": {
			Source:         `{objects(first: 10) { int objects(first: 5) { int } }}`,
			ExpectedCost:   1 + 10*(2+5),
			MaxCost:        10,
			ExpectedErrors: 1,
		},
		"FragmentSpreads": {
			Source:       `{objects(first: 10) { ...f }} fragment f on Object {... on Object {a: int b: int}}`,
			ExpectedCost: 1 + 10*2,
			MaxCost:      100,
		},
		"DefaultArg": {
			Source:       `{costFromArg}`,
			ExpectedCost: 10,
			MaxCost:      100,
		},
		"QueryArg": {
			Source:       `query Foo($cost: Int) {costFromArg(cost: $cost)}`,
			ExpectedCost: 20,
			VariableValues: map[string]interface{}{
				"cost": 20,
			},
			MaxCost: 100,
		},
		"MultipleMatchingOperations": {
			Source:         `query Foo {int} query Foo {int}`,
			ExpectedErrors: 1,
		},
		"Overflow": {
			Source: `{objects(first: 10)
					 {objects(first: 10)
					 {objects(first: 10)
					 {objects(first: 10)
					 {objects(first: 10)
					 {objects(first: 10)
					 {objects(first: 10)
					 {objects(first: 10)
					 {objects(first: 10)
					 {objects(first: 10)
					 {objects(first: 10)
					 {objects(first: 10)
					 {objects(first: 10)
					 {objects(first: 10)
					 {objects(first: 10)
					 {objects(first: 10)
					 {objects(first: 10)
					 {objects(first: 10)
					 {objects(first: 10)
					 {objects(first: 10)
					 {objects(first: 10)
					 {int}
					 }}}}}}}}}}}}}}}}}}}}}`,
			ExpectedCost:   maxInt,
			MaxCost:        10,
			ExpectedErrors: 1,
		},
	} {
		t.Run(name, func(t *testing.T) {
			doc, parseErrs := parser.ParseDocument([]byte(tc.Source))
			require.Empty(t, parseErrs)
			require.NotNil(t, doc)

			var cost int
			errs := ValidateDocument(doc, s, ValidateCost(tc.OperationName, tc.VariableValues, tc.MaxCost, &cost, schema.FieldCost{Resolver: 1}))
			for _, err := range errs {
				assert.NotEmpty(t, err.Message)
				assert.NotEmpty(t, err.Locations)
				assert.False(t, err.isSecondary)
			}
			assert.Equal(t, tc.ExpectedCost, cost)
			assert.Len(t, errs, tc.ExpectedErrors)
		})
	}
}
