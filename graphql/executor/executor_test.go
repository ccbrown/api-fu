package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ccbrown/api-fu/graphql/parser"
	"github.com/ccbrown/api-fu/graphql/schema"
	"github.com/ccbrown/api-fu/graphql/schema/introspection"
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
	IsTypeOf: func(v interface{}) bool {
		_, ok := v.(dog)
		return ok
	},
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
	IsTypeOf: func(v interface{}) bool {
		_, ok := v.(cat)
		return ok
	},
}

var objectType = &schema.ObjectType{
	Name: "Object",
}

type object struct {
	Error error
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
				return &object{}, nil
			},
		},
		"objectsWithError": &schema.FieldDefinition{
			Type: schema.NewListType(objectType),
			Resolve: func(*schema.FieldContext) (interface{}, error) {
				return []*object{&object{}, &object{Error: fmt.Errorf("error")}}, nil
			},
		},
		"intOneOrError": &schema.FieldDefinition{
			Type: schema.IntType,
			Resolve: func(ctx *schema.FieldContext) (interface{}, error) {
				if err := ctx.Object.(*object).Error; err != nil {
					return nil, err
				}
				return 1, nil
			},
		},
		"error": &schema.FieldDefinition{
			Type: schema.IntType,
			Resolve: func(*schema.FieldContext) (interface{}, error) {
				return nil, fmt.Errorf("error")
			},
		},
		"nonNullError": &schema.FieldDefinition{
			Type: schema.NewNonNullType(schema.IntType),
			Resolve: func(*schema.FieldContext) (interface{}, error) {
				return nil, fmt.Errorf("error")
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

func TestSubscribe(t *testing.T) {
	s, err := schema.New(&schema.SchemaDefinition{
		Query: objectType,
		Subscription: &schema.ObjectType{
			Name: "Subscription",
			Fields: map[string]*schema.FieldDefinition{
				"int": &schema.FieldDefinition{
					Type: schema.NewNonNullType(schema.IntType),
					Resolve: func(*schema.FieldContext) (interface{}, error) {
						return 1, nil
					},
				},
			},
		},
		AdditionalTypes: []schema.NamedType{dogType, catType},
	})
	require.NoError(t, err)
	doc, parseErrs := parser.ParseDocument([]byte(`subscription {int}`))
	require.Empty(t, parseErrs)
	require.Empty(t, validator.ValidateDocument(doc, s))

	assert.True(t, IsSubscription(doc, ""))

	r := &Request{
		Document: doc,
		Schema:   s,
	}

	responseStream, err := Subscribe(context.Background(), r)
	assert.NoError(t, err)
	assert.Equal(t, 1, responseStream)

	data, errs := ExecuteRequest(context.Background(), r)
	assert.Empty(t, errs)
	assert.Equal(t, 1, data.Len())
}

func TestExecuteRequest(t *testing.T) {
	s, err := schema.New(&schema.SchemaDefinition{
		Query:    objectType,
		Mutation: mutationType,
		Directives: map[string]*schema.DirectiveDefinition{
			"include": schema.IncludeDirective,
			"skip":    schema.SkipDirective,
		},
		AdditionalTypes: []schema.NamedType{dogType, catType},
	})
	require.NoError(t, err)

	t.Run("IntrospectionQuery", func(t *testing.T) {
		parsed, parseErrs := parser.ParseDocument(introspection.Query)
		require.Empty(t, parseErrs)
		require.Empty(t, validator.ValidateDocument(parsed, s))
		_, errs := ExecuteRequest(context.Background(), &Request{
			Document: parsed,
			Schema:   s,
		})
		require.Empty(t, errs)
	})

	for name, tc := range map[string]struct {
		Document       string
		ExpectedData   string
		ExpectedErrors []*Error
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
		"Error": {
			Document:     `{error error}`,
			ExpectedData: `{"error":null}`,
			ExpectedErrors: []*Error{
				&Error{
					Locations: []Location{{1, 2}, {1, 8}},
					Path:      []interface{}{"error"},
				},
			},
		},
		"PropagatedError": {
			Document:     `{object{nonNullError}}`,
			ExpectedData: `{"object":null}`,
			ExpectedErrors: []*Error{
				&Error{
					Locations: []Location{{1, 9}},
					Path:      []interface{}{"object", "nonNullError"},
				},
			},
		},
		"ListError": {
			Document:     `{objs:objectsWithError{n:intOneOrError}}`,
			ExpectedData: `{"objs":[{"n":1},{"n":null}]}`,
			ExpectedErrors: []*Error{
				&Error{
					Locations: []Location{{1, 24}},
					Path:      []interface{}{"objs", 1, "n"},
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			parsed, parseErrs := parser.ParseDocument([]byte(tc.Document))
			require.Empty(t, parseErrs)
			require.Empty(t, validator.ValidateDocument(parsed, s))
			data, errs := ExecuteRequest(context.Background(), &Request{
				Document:       parsed,
				Schema:         s,
				VariableValues: tc.VariableValues,
			})
			serializedData, err := json.Marshal(data)
			require.NoError(t, err)
			assert.Equal(t, tc.ExpectedData, string(serializedData))

			serializedErrors, err := json.Marshal(errs)
			require.NoError(t, err)

			if len(tc.ExpectedErrors) == 0 {
				assert.Empty(t, errs)
			} else {
				assert.Len(t, errs, len(tc.ExpectedErrors))
				for _, expected := range tc.ExpectedErrors {
					matched := false
					for _, actual := range errs {
						if reflect.DeepEqual(actual.Locations, expected.Locations) && reflect.DeepEqual(actual.Path, expected.Path) {
							matched = true
							break
						}
					}
					assert.True(t, matched, "couldn't find %+v in %v", *expected, string(serializedErrors))
				}
			}
		})
	}
}

func TestGetOperation(t *testing.T) {
	doc, errs := parser.ParseDocument([]byte(`{x} {x} query q {x} mutation m {x} mutation m {x}`))
	assert.Empty(t, errs)

	_, err := getOperation(doc, "")
	assert.NotNil(t, err)

	op, err := getOperation(doc, "m")
	assert.Nil(t, op)
	assert.NotNil(t, err)

	op, err = getOperation(doc, "q")
	assert.NotNil(t, op)
	assert.Nil(t, err)
}
