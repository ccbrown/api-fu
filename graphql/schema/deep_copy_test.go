package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var petType = &InterfaceType{
	Name: "Pet",
	Fields: map[string]*FieldDefinition{
		"nickname": {
			Type: StringType,
		},
	},
}

var dogType = &ObjectType{
	Name: "Dog",
	Fields: map[string]*FieldDefinition{
		"nickname": {
			Type: StringType,
		},
		"barkVolume": {
			Type: IntType,
		},
	},
	ImplementedInterfaces: []*InterfaceType{petType},
	IsTypeOf:              func(interface{}) bool { return false },
}

var fooBarEnumType = &EnumType{
	Name: "FooBarEnum",
	Values: map[string]*EnumValueDefinition{
		"FOO": {},
		"BAR": {},
	},
}

var objectType = &ObjectType{
	Name: "Object",
	Fields: map[string]*FieldDefinition{
		"pet": {
			Type: petType,
			Arguments: map[string]*InputValueDefinition{
				"booleanArg": {
					Type: BooleanType,
				},
			},
		},
		"union": {
			Type: &UnionType{
				Name: "Union",
				MemberTypes: []*ObjectType{
					{
						Name: "UnionObjectA",
						Fields: map[string]*FieldDefinition{
							"a": {
								Type: StringType,
							},
							"scalar": {
								Type: StringType,
							},
						},
						IsTypeOf: func(interface{}) bool { return false },
					},
					{
						Name: "UnionObjectB",
						Fields: map[string]*FieldDefinition{
							"b": {
								Type: StringType,
							},
							"scalar": {
								Type: StringType,
							},
						},
						IsTypeOf: func(interface{}) bool { return false },
					},
				},
			},
		},
		"int": {
			Type: IntType,
		},
		"nonNullInt": {
			Type: NewNonNullType(IntType),
		},
		"enum": {
			Type: fooBarEnumType,
		},
	},
}

func TestDeepCopySchemaDefinition(t *testing.T) {
	def := &SchemaDefinition{
		Query: objectType,
		Directives: map[string]*DirectiveDefinition{
			"directive": {
				Locations: []DirectiveLocation{DirectiveLocationField, DirectiveLocationFragmentSpread, DirectiveLocationInlineFragment},
			},
		},
		AdditionalTypes: []NamedType{dogType},
	}

	getType := func(def *SchemaDefinition, name string) Type {
		var ret Type
		Inspect(def, func(node any) bool {
			if t, ok := node.(NamedType); ok && t.TypeName() == name {
				ret = t
				return false
			}
			return true
		})
		return ret
	}

	newTypeDescriptions := map[string]string{
		"Pet":          "new pet description",
		"Dog":          "new dog description",
		"FooBarEnum":   "new enum description",
		"Union":        "new union description",
		"UnionObjectA": "new union object a description",
		"UnionObjectB": "new union object b description",
	}

	// Make a copy of the definition and modify all the types.
	defCopy := def.Clone()
	for name, desc := range newTypeDescriptions {
		switch typ := getType(defCopy, name).(type) {
		case *ObjectType:
			typ.Description = desc
		case *EnumType:
			typ.Description = desc
		case *UnionType:
			typ.Description = desc
		case *InterfaceType:
			typ.Description = desc
		default:
			t.Fatalf("unexpected type %T for %v", typ, name)
		}
	}

	// Make sure the copy is still valid.
	_, err := New(defCopy)
	require.NoError(t, err)

	// Make sure the original definition is unchanged.
	for name := range newTypeDescriptions {
		switch typ := getType(def, name).(type) {
		case *ObjectType:
			assert.Empty(t, typ.Description)
		case *EnumType:
			assert.Empty(t, typ.Description)
		case *UnionType:
			assert.Empty(t, typ.Description)
		case *InterfaceType:
			assert.Empty(t, typ.Description)
		default:
			t.Fatalf("unexpected type %T for %v", typ, name)
		}
	}

	// Make sure the original is still valid.
	_, err = New(def)
	require.NoError(t, err)
}
