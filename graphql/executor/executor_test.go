package executor

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ccbrown/api-fu/graphql/parser"
	"github.com/ccbrown/api-fu/graphql/schema"
	"github.com/ccbrown/api-fu/graphql/validator"
)

var petType = &schema.InterfaceType{
	Name: "Pet",
	Fields: map[string]*schema.FieldDefinition{
		"nickname": &schema.FieldDefinition{
			Type: schema.StringType,
		},
	},
}

type dog struct{}
type cat struct{}

func init() {
	petType.ObjectType = func(v interface{}) *schema.ObjectType {
		switch v.(type) {
		case dog:
			return dogType
		case cat:
			return catType
		}
		return nil
	}
}

var dogType = &schema.ObjectType{
	Name: "Dog",
	Fields: map[string]*schema.FieldDefinition{
		"nickname": &schema.FieldDefinition{
			Type: schema.StringType,
			Resolve: func(*schema.FieldContext) (interface{}, error) {
				return "fido", nil
			},
		},
		"barkVolume": &schema.FieldDefinition{
			Type: schema.IntType,
			Resolve: func(*schema.FieldContext) (interface{}, error) {
				return 10, nil
			},
		},
	},
	ImplementedInterfaces: []*schema.InterfaceType{petType},
}

var catType = &schema.ObjectType{
	Name: "Cat",
	Fields: map[string]*schema.FieldDefinition{
		"nickname": &schema.FieldDefinition{
			Type: schema.StringType,
			Resolve: func(*schema.FieldContext) (interface{}, error) {
				return "fluffy", nil
			},
		},
		"meowVolume": &schema.FieldDefinition{
			Type: schema.IntType,
			Resolve: func(*schema.FieldContext) (interface{}, error) {
				return 10, nil
			},
		},
	},
	ImplementedInterfaces: []*schema.InterfaceType{petType},
}

var objectType = &schema.ObjectType{
	Name: "Object",
}

func init() {
	objectType.Fields = map[string]*schema.FieldDefinition{
		"intOne": &schema.FieldDefinition{
			Type: schema.IntType,
			Resolve: func(*schema.FieldContext) (interface{}, error) {
				return 1, nil
			},
		},
		"pet": &schema.FieldDefinition{
			Type: petType,
			Resolve: func(*schema.FieldContext) (interface{}, error) {
				return dog{}, nil
			},
		},
		"intTwo": &schema.FieldDefinition{
			Type: schema.IntType,
			Resolve: func(*schema.FieldContext) (interface{}, error) {
				return 2, nil
			},
		},
		"stringFoo": &schema.FieldDefinition{
			Type: schema.StringType,
			Resolve: func(*schema.FieldContext) (interface{}, error) {
				return "foo", nil
			},
		},
		"object": &schema.FieldDefinition{
			Type: objectType,
			Resolve: func(*schema.FieldContext) (interface{}, error) {
				return struct{}{}, nil
			},
		},
	}
}

var theNumber int
var mutationType = &schema.ObjectType{
	Name: "Mutation",
	Fields: map[string]*schema.FieldDefinition{
		"changeTheNumber": &schema.FieldDefinition{
			Type: &schema.ObjectType{
				Name: "ChangeTheNumberResult",
				Fields: map[string]*schema.FieldDefinition{
					"theNumber": &schema.FieldDefinition{
						Type: schema.NewNonNullType(schema.IntType),
						Resolve: func(*schema.FieldContext) (interface{}, error) {
							return theNumber, nil
						},
					},
				},
			},
			Arguments: map[string]*schema.InputValueDefinition{
				"newNumber": &schema.InputValueDefinition{
					Type: schema.NewNonNullType(schema.IntType),
				},
			},
			Resolve: func(ctx *schema.FieldContext) (interface{}, error) {
				theNumber = ctx.Arguments["newNumber"].(int)
				return struct{}{}, nil
			},
		},
	},
}

func TestExecuteRequest(t *testing.T) {
	s, err := schema.New(&schema.SchemaDefinition{
		Query:        objectType,
		Mutation:     mutationType,
		Subscription: objectType,
		DirectiveDefinitions: map[string]*schema.DirectiveDefinition{
			"include": schema.IncludeDirective,
			"skip":    schema.SkipDirective,
		},
		AdditionalTypes: []schema.NamedType{dogType, catType},
	})
	require.NoError(t, err)

	for name, tc := range map[string]struct {
		Document       string
		ExpectedData   string
		VariableValues map[string]interface{}
	}{
		"Query": {
			Document:     `{intOne stringFoo object {intOne}}`,
			ExpectedData: `{"intOne":1,"stringFoo":"foo","object":{"intOne":1}}`,
		},
		"SkipTrue": {
			Document:     `{intOne @skip(if: true)}`,
			ExpectedData: `{}`,
		},
		"SkipFalse": {
			Document:     `{intOne @skip(if: false)}`,
			ExpectedData: `{"intOne":1}`,
		},
		"IncludeTrue": {
			Document:     `{intOne @include(if: true)}`,
			ExpectedData: `{"intOne":1}`,
		},
		"IncludeFalse": {
			Document:     `{intOne @include(if: false)}`,
			ExpectedData: `{}`,
		},
		"InlineFragmentCollection": {
			Document:     `{...{intOne} ...{intOne}}`,
			ExpectedData: `{"intOne":1}`,
		},
		"FragmentCollection": {
			Document:     `{object{intOne} ...Frag} fragment Frag on Object {object{stringFoo} intTwo}`,
			ExpectedData: `{"object":{"intOne":1,"stringFoo":"foo"},"intTwo":2}`,
		},
		"Mutation": {
			Document:     `mutation {changeTheNumber(newNumber: 1) {theNumber}}`,
			ExpectedData: `{"changeTheNumber":{"theNumber":1}}`,
		},
		"SerialMutation": {
			Document: `mutation {
				first: changeTheNumber(newNumber: 1) {theNumber}
				second: changeTheNumber(newNumber: 3) {theNumber}
				third: changeTheNumber(newNumber: 2) {theNumber}
			}`,
			ExpectedData: `{"first":{"theNumber":1},"second":{"theNumber":3},"third":{"theNumber":2}}`,
		},
		"Variable": {
			Document:     `mutation ($n: Int!) {changeTheNumber(newNumber: $n) {theNumber}}`,
			ExpectedData: `{"changeTheNumber":{"theNumber":1}}`,
			VariableValues: map[string]interface{}{
				"n": 1,
			},
		},
		"VariableDefault": {
			Document:     `mutation ($n: Int! = 1) {changeTheNumber(newNumber: $n) {theNumber}}`,
			ExpectedData: `{"changeTheNumber":{"theNumber":1}}`,
		},
		"ObjectFragmentSpread": {
			Document:     `{pet{... on Cat{meowVolume} ... on Dog{barkVolume}}}`,
			ExpectedData: `{"pet":{"barkVolume":10}}`,
		},
		"InterfaceFragmentSpread": {
			Document:     `{pet{... on Pet{nickname}}}`,
			ExpectedData: `{"pet":{"nickname":"fido"}}`,
		},
		"InterfaceTypename": {
			Document:     `{pet{__typename}}`,
			ExpectedData: `{"pet":{"__typename":"Dog"}}`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			parsed, errs := parser.ParseDocument([]byte(tc.Document))
			require.Empty(t, errs)
			require.Empty(t, validator.ValidateDocument(parsed, s))
			response := ExecuteRequest(&Request{
				Document:       parsed,
				Schema:         s,
				VariableValues: tc.VariableValues,
			})
			serializedData, err := json.Marshal(response.Data)
			require.NoError(t, err)
			assert.Equal(t, tc.ExpectedData, string(serializedData))
		})
	}
}

func TestGetOperation(t *testing.T) {
	doc, errs := parser.ParseDocument([]byte(`{x} {x} query q {x} mutation m {x} mutation m {x}`))
	assert.Empty(t, errs)

	op, err := getOperation(doc, "")
	assert.NotEmpty(t, err)

	op, err = getOperation(doc, "m")
	assert.NotEmpty(t, err)

	op, err = getOperation(doc, "q")
	assert.NotNil(t, op)
	assert.Empty(t, err)
}
