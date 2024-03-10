package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

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
		"nickname": {
			Type: schema.StringType,
		},
	},
}

type dog struct{}
type cat struct{}

var dogType = &schema.ObjectType{
	Name: "Dog",
	Fields: map[string]*schema.FieldDefinition{
		"nickname": {
			Type: schema.StringType,
			Resolve: func(schema.FieldContext) (interface{}, error) {
				return "fido", nil
			},
		},
		"barkVolume": {
			Type: schema.IntType,
			Resolve: func(schema.FieldContext) (interface{}, error) {
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
		"nickname": {
			Type: schema.StringType,
			Resolve: func(schema.FieldContext) (interface{}, error) {
				return "fluffy", nil
			},
		},
		"meowVolume": {
			Type: schema.IntType,
			Resolve: func(schema.FieldContext) (interface{}, error) {
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

var stringPromises []ResolvePromise

func init() {
	objectType.Fields = map[string]*schema.FieldDefinition{
		"intOne": {
			Type: schema.IntType,
			Resolve: func(schema.FieldContext) (interface{}, error) {
				return 1, nil
			},
		},
		"pet": {
			Type: petType,
			Resolve: func(schema.FieldContext) (interface{}, error) {
				return dog{}, nil
			},
		},
		"intTwo": {
			Type: schema.IntType,
			Resolve: func(schema.FieldContext) (interface{}, error) {
				return 2, nil
			},
		},
		"asyncString": {
			Type: schema.StringType,
			Resolve: func(schema.FieldContext) (interface{}, error) {
				ch := make(ResolvePromise, 1)
				stringPromises = append(stringPromises, ch)
				return ResolvePromise(ch), nil
			},
		},
		"stringFoo": {
			Type: schema.StringType,
			Resolve: func(schema.FieldContext) (interface{}, error) {
				return "foo", nil
			},
		},
		"object": {
			Type: objectType,
			Resolve: func(schema.FieldContext) (interface{}, error) {
				return &object{}, nil
			},
		},
		"nonNullIntListWithNull": {
			Type: schema.NewListType(schema.NewNonNullType(schema.IntType)),
			Resolve: func(schema.FieldContext) (interface{}, error) {
				return []interface{}{1, nil, 3}, nil
			},
		},
		"objectsWithError": {
			Type: schema.NewListType(objectType),
			Resolve: func(schema.FieldContext) (interface{}, error) {
				return []*object{{}, {Error: fmt.Errorf("error")}, {}}, nil
			},
		},
		"intOneOrError": {
			Type: schema.IntType,
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				if err := ctx.Object.(*object).Error; err != nil {
					return nil, err
				}
				return 1, nil
			},
		},
		"error": {
			Type: schema.IntType,
			Resolve: func(schema.FieldContext) (interface{}, error) {
				return nil, fmt.Errorf("error")
			},
		},
		"nonNullError": {
			Type: schema.NewNonNullType(schema.IntType),
			Resolve: func(schema.FieldContext) (interface{}, error) {
				return nil, fmt.Errorf("error")
			},
		},
		"badResolveValue": {
			Type: schema.IntType,
			Resolve: func(schema.FieldContext) (interface{}, error) {
				return &struct{}{}, nil
			},
		},
		"intListWithBadResolveValue": {
			Type: schema.NewListType(schema.IntType),
			Resolve: func(schema.FieldContext) (interface{}, error) {
				return []interface{}{1, &struct{}{}, 3}, nil
			},
		},
	}
}

var theNumber int
var mutationType = &schema.ObjectType{
	Name: "Mutation",
	Fields: map[string]*schema.FieldDefinition{
		"asyncString": {
			Type: schema.StringType,
			Resolve: func(schema.FieldContext) (interface{}, error) {
				ch := make(ResolvePromise, 1)
				stringPromises = append(stringPromises, ch)
				return ResolvePromise(ch), nil
			},
		},
		"changeTheNumber": {
			Type: &schema.ObjectType{
				Name: "ChangeTheNumberResult",
				Fields: map[string]*schema.FieldDefinition{
					"theNumber": {
						Type: schema.NewNonNullType(schema.IntType),
						Resolve: func(schema.FieldContext) (interface{}, error) {
							return theNumber, nil
						},
					},
				},
			},
			Arguments: map[string]*schema.InputValueDefinition{
				"newNumber": {
					Type: schema.NewNonNullType(schema.IntType),
				},
			},
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
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
				"int": {
					Type: schema.NewNonNullType(schema.IntType),
					Resolve: func(schema.FieldContext) (interface{}, error) {
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
	require.Empty(t, validator.ValidateDocument(doc, s, nil))

	assert.True(t, IsSubscription(doc, ""))

	r := &Request{
		Document: doc,
		Schema:   s,
	}

	responseStream, err := Subscribe(context.Background(), r)
	assert.Nil(t, err)
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
		require.Empty(t, validator.ValidateDocument(parsed, s, nil))
		_, errs := ExecuteRequest(context.Background(), &Request{
			Document: parsed,
			Schema:   s,
		})
		require.Empty(t, errs)
	})

	for name, tc := range map[string]struct {
		Document             string
		ExpectedData         string
		ExpectedErrors       []*Error
		ExpectedIdlePromises []int
		VariableValues       map[string]interface{}
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
		"BadResolveValue": {
			Document:     `{intOne badResolveValue}`,
			ExpectedData: `{"intOne":1,"badResolveValue":null}`,
			ExpectedErrors: []*Error{
				{
					Locations: []Location{{1, 9}},
					Path:      []interface{}{"badResolveValue"},
				},
			},
		},
		"IntListWithBadResolveValue": {
			Document:     `{intOne l:intListWithBadResolveValue}`,
			ExpectedData: `{"intOne":1,"l":[1,null,3]}`,
			ExpectedErrors: []*Error{
				{
					Locations: []Location{{1, 9}},
					Path:      []interface{}{"l", 1},
				},
			},
		},
		"InlineFragmentCollection": {
			Document:     `{...{intOne} ...{intOne}}`,
			ExpectedData: `{"intOne":1}`,
		},
		"FragmentCollection": {
			Document:     `{object{intOne} ...Frag} fragment Frag on Object {object{stringFoo} intTwo}`,
			ExpectedData: `{"object":{"intOne":1,"stringFoo":"foo"},"intTwo":2}`,
		},
		"AsyncQuery": {
			Document:             `{a:asyncString b:asyncString}`,
			ExpectedData:         `{"a":"s","b":"s"}`,
			ExpectedIdlePromises: []int{2},
		},
		"AsyncQueryNested": {
			Document:             `{a:asyncString object{b:asyncString}}`,
			ExpectedData:         `{"a":"s","object":{"b":"s"}}`,
			ExpectedIdlePromises: []int{2},
		},
		"AsyncMutation": {
			Document:             `mutation {a:asyncString b:asyncString}`,
			ExpectedData:         `{"a":"s","b":"s"}`,
			ExpectedIdlePromises: []int{1, 1},
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
				{
					Locations: []Location{{1, 2}, {1, 8}},
					Path:      []interface{}{"error"},
				},
			},
		},
		"PropagatedError": {
			Document:     `{object{nonNullError}}`,
			ExpectedData: `{"object":null}`,
			ExpectedErrors: []*Error{
				{
					Locations: []Location{{1, 9}},
					Path:      []interface{}{"object", "nonNullError"},
				},
			},
		},
		"ListError": {
			Document:     `{object{object{object{object{objs:objectsWithError{n:intOneOrError}}}}}}`,
			ExpectedData: `{"object":{"object":{"object":{"object":{"objs":[{"n":1},{"n":null},{"n":1}]}}}}}`,
			ExpectedErrors: []*Error{
				{
					Locations: []Location{{1, 52}},
					Path:      []interface{}{"object", "object", "object", "object", "objs", 1, "n"},
				},
			},
		},
		"NonNullIntListWithNull": {
			Document:     `{l:nonNullIntListWithNull}`,
			ExpectedData: `{"l":null}`,
			ExpectedErrors: []*Error{
				{
					Locations: []Location{{1, 2}},
					Path:      []interface{}{"l", 1},
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			parsed, parseErrs := parser.ParseDocument([]byte(tc.Document))
			require.Empty(t, parseErrs)
			require.Empty(t, validator.ValidateDocument(parsed, s, nil))
			data, errs := ExecuteRequest(context.Background(), &Request{
				Document:       parsed,
				Schema:         s,
				VariableValues: tc.VariableValues,
				IdleHandler: func() {
					require.NotEmpty(t, tc.ExpectedIdlePromises)
					assert.Len(t, stringPromises, tc.ExpectedIdlePromises[len(tc.ExpectedIdlePromises)-1])
					for _, p := range stringPromises {
						p <- ResolveResult{
							Value: "s",
						}
					}
					stringPromises = nil
					tc.ExpectedIdlePromises = tc.ExpectedIdlePromises[:len(tc.ExpectedIdlePromises)-1]
				},
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

	_, err := GetOperation(doc, "")
	assert.NotNil(t, err)

	op, err := GetOperation(doc, "m")
	assert.Nil(t, op)
	assert.NotNil(t, err)

	op, err = GetOperation(doc, "q")
	assert.NotNil(t, op)
	assert.Nil(t, err)

	doc, errs = parser.ParseDocument([]byte(`query q {x}`))
	assert.Empty(t, errs)

	op, err = GetOperation(doc, "")
	assert.NotNil(t, op)
	assert.Nil(t, err)
}

var sink interface{}

func BenchmarkExecuteRequest(b *testing.B) {
	var objectType = &schema.ObjectType{
		Name: "Object",
	}

	objectType.Fields = map[string]*schema.FieldDefinition{
		"string": {
			Type: schema.StringType,
			Resolve: func(schema.FieldContext) (interface{}, error) {
				return "foo", nil
			},
		},
		"nonNullString": {
			Type: schema.NewNonNullType(schema.StringType),
			Resolve: func(schema.FieldContext) (interface{}, error) {
				return "foo", nil
			},
		},
		"objects": {
			Type: schema.NewListType(objectType),
			Arguments: map[string]*schema.InputValueDefinition{
				"count": {
					Type: schema.NewNonNullType(schema.IntType),
				},
			},
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				return make([]struct{}, ctx.Arguments["count"].(int)), nil
			},
		},
	}

	s, err := schema.New(&schema.SchemaDefinition{
		Query: objectType,
	})
	require.NoError(b, err)
	doc, parseErrs := parser.ParseDocument([]byte(`{
		string
		objects(count: 20) {
			string
			nonNullString
			objects(count: 100) {
				string
				nonNullString
			}
		}
	}`))
	require.Empty(b, parseErrs)
	require.Empty(b, validator.ValidateDocument(doc, s, nil))

	r := &Request{
		Document: doc,
		Schema:   s,
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sink, _ = ExecuteRequest(context.Background(), r)
	}
}

func TestContextCancelation(t *testing.T) {
	var objectType = &schema.ObjectType{
		Name: "Object",
	}

	objectType.Fields = map[string]*schema.FieldDefinition{
		"slowString": {
			Type: schema.StringType,
			Resolve: func(schema.FieldContext) (interface{}, error) {
				time.Sleep(100 * time.Millisecond)
				return "foo", nil
			},
		},
		"objects": {
			Type: schema.NewListType(objectType),
			Arguments: map[string]*schema.InputValueDefinition{
				"count": {
					Type: schema.NewNonNullType(schema.IntType),
				},
			},
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				return make([]struct{}, ctx.Arguments["count"].(int)), nil
			},
		},
	}

	s, err := schema.New(&schema.SchemaDefinition{
		Query: objectType,
	})
	require.NoError(t, err)
	doc, parseErrs := parser.ParseDocument([]byte(`{
		objects(count: 100) {
			slowString
		}
	}`))
	require.Empty(t, parseErrs)
	require.Empty(t, validator.ValidateDocument(doc, s, nil))

	r := &Request{
		Document: doc,
		Schema:   s,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	startTime := time.Now()
	_, errs := ExecuteRequest(ctx, r)
	// The request should be cancelled early.
	assert.Less(t, time.Since(startTime), 2*time.Second)
	assert.NotEmpty(t, errs)
}
