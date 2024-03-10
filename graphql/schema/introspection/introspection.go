package introspection

import (
	"fmt"

	"github.com/ccbrown/api-fu/graphql/schema"
)

var NamedTypes = map[string]schema.NamedType{
	"__Schema":            SchemaType,
	"__Type":              TypeType,
	"__Field":             FieldType,
	"__InputValue":        InputValueType,
	"__EnumValue":         EnumValueType,
	"__TypeKind":          TypeKindType,
	"__Directive":         DirectiveType,
	"__DirectiveLocation": DirectiveLocationType,
}

var MetaFields = map[string]*schema.FieldDefinition{
	"__schema": {
		Type: schema.NewNonNullType(SchemaType),
		Cost: schema.FieldResolverCost(0),
		Resolve: func(ctx schema.FieldContext) (interface{}, error) {
			return ctx.Schema, nil
		},
	},
	"__type": {
		Type: TypeType,
		Cost: schema.FieldResolverCost(0),
		Arguments: map[string]*schema.InputValueDefinition{
			"name": {
				Type: schema.NewNonNullType(schema.StringType),
			},
		},
		Resolve: func(ctx schema.FieldContext) (interface{}, error) {
			return ctx.Schema.NamedTypes()[ctx.Arguments["name"].(string)], nil
		},
	},
}

func nullableString(s string) (interface{}, error) {
	if s != "" {
		return s, nil
	}
	return nil, nil
}

func inputValues(values map[string]*schema.InputValueDefinition) (interface{}, error) {
	ret := []inputValue{}
	for name, def := range values {
		ret = append(ret, inputValue{
			Name:       name,
			Definition: def,
		})
	}
	return ret, nil
}

type directive struct {
	Name       string
	Definition *schema.DirectiveDefinition
}

var SchemaType = &schema.ObjectType{
	Name: "__Schema",
	Fields: map[string]*schema.FieldDefinition{
		"types": {
			Type: schema.NewNonNullType(schema.NewListType(schema.NewNonNullType(TypeType))),
			Cost: schema.FieldResolverCost(0),
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				namedTypes := ctx.Schema.NamedTypes()
				ret := make([]schema.Type, len(namedTypes))
				i := 0
				for _, def := range namedTypes {
					ret[i] = def
					i++
				}
				return ret, nil
			},
		},
		"queryType": {
			Type: schema.NewNonNullType(TypeType),
			Cost: schema.FieldResolverCost(0),
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				return ctx.Schema.QueryType(), nil
			},
		},
		"mutationType": {
			Type: TypeType,
			Cost: schema.FieldResolverCost(0),
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				return ctx.Schema.MutationType(), nil
			},
		},
		"subscriptionType": {
			Type: TypeType,
			Cost: schema.FieldResolverCost(0),
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				return ctx.Schema.SubscriptionType(), nil
			},
		},
		"directives": {
			Type: schema.NewNonNullType(schema.NewListType(schema.NewNonNullType(DirectiveType))),
			Cost: schema.FieldResolverCost(0),
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				directives := ctx.Schema.Directives()
				ret := make([]directive, len(directives))
				i := 0
				for name, def := range directives {
					ret[i] = directive{
						Name:       name,
						Definition: def,
					}
					i++
				}
				return ret, nil
			},
		},
	},
}

type typeKind string

const (
	typeKindScalar      typeKind = "scalar"
	typeKindObject      typeKind = "object"
	typeKindInterface   typeKind = "interface"
	typeKindUnion       typeKind = "union"
	typeKindEnum        typeKind = "enum"
	typeKindInputObject typeKind = "input_object"
	typeKindList        typeKind = "list"
	typeKindNonNull     typeKind = "non_null"
)

var TypeKindType = &schema.EnumType{
	Name: "__TypeKind",
	Values: map[string]*schema.EnumValueDefinition{
		"SCALAR": {
			Value: typeKindScalar,
		},
		"OBJECT": {
			Value: typeKindObject,
		},
		"INTERFACE": {
			Value: typeKindInterface,
		},
		"UNION": {
			Value: typeKindUnion,
		},
		"ENUM": {
			Value: typeKindEnum,
		},
		"INPUT_OBJECT": {
			Value: typeKindInputObject,
		},
		"LIST": {
			Value: typeKindList,
		},
		"NON_NULL": {
			Value: typeKindNonNull,
		},
	},
}

var TypeType = &schema.ObjectType{
	Name: "__Type",
}

func init() {
	TypeType.Fields = map[string]*schema.FieldDefinition{
		"kind": {
			Type: schema.NewNonNullType(TypeKindType),
			Cost: schema.FieldResolverCost(0),
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				switch t := ctx.Object.(type) {
				case *schema.ScalarType:
					return typeKindScalar, nil
				case *schema.ObjectType:
					return typeKindObject, nil
				case *schema.InterfaceType:
					return typeKindInterface, nil
				case *schema.UnionType:
					return typeKindUnion, nil
				case *schema.EnumType:
					return typeKindEnum, nil
				case *schema.InputObjectType:
					return typeKindInputObject, nil
				case *schema.ListType:
					return typeKindList, nil
				case *schema.NonNullType:
					return typeKindNonNull, nil
				default:
					return nil, fmt.Errorf(fmt.Sprintf("unexpected type: %T", t))
				}
			},
		},
		"name": {
			Type: schema.StringType,
			Cost: schema.FieldResolverCost(0),
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				if t, ok := ctx.Object.(schema.NamedType); ok {
					return t.TypeName(), nil
				}
				return nil, nil
			},
		},
		"description": {
			Type: schema.StringType,
			Cost: schema.FieldResolverCost(0),
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				description := ""
				switch t := ctx.Object.(type) {
				case *schema.ScalarType:
					description = t.Description
				case *schema.ObjectType:
					description = t.Description
				case *schema.InterfaceType:
					description = t.Description
				case *schema.UnionType:
					description = t.Description
				case *schema.EnumType:
					description = t.Description
				case *schema.InputObjectType:
					description = t.Description
				}
				return nullableString(description)
			},
		},
		"fields": {
			Type: schema.NewListType(schema.NewNonNullType(FieldType)),
			Cost: schema.FieldResolverCost(0),
			Arguments: map[string]*schema.InputValueDefinition{
				"includeDeprecated": {
					Type:         schema.BooleanType,
					DefaultValue: false,
				},
			},
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				var fields map[string]*schema.FieldDefinition
				switch t := ctx.Object.(type) {
				case *schema.ObjectType:
					fields = t.Fields
				case *schema.InterfaceType:
					fields = t.Fields
				default:
					return nil, nil
				}
				includeDeprecated := ctx.Arguments["includeDeprecated"].(bool)
				ret := []field{}
				for name, def := range fields {
					if (def.DeprecationReason == "" || includeDeprecated) && def.RequiredFeatures.IsSubsetOf(ctx.Features) {
						ret = append(ret, field{
							Name:       name,
							Definition: def,
						})
					}
				}
				return ret, nil
			},
		},
		"interfaces": {
			Type: schema.NewListType(schema.NewNonNullType(TypeType)),
			Cost: schema.FieldResolverCost(0),
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				if t, ok := ctx.Object.(*schema.ObjectType); ok {
					return t.ImplementedInterfaces, nil
				}
				return nil, nil
			},
		},
		"possibleTypes": {
			Type: schema.NewListType(schema.NewNonNullType(TypeType)),
			Cost: schema.FieldResolverCost(0),
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				switch t := ctx.Object.(type) {
				case *schema.InterfaceType:
					return ctx.Schema.InterfaceImplementations(t.Name), nil
				case *schema.UnionType:
					return t.MemberTypes, nil
				default:
					return nil, nil
				}
			},
		},
		"enumValues": {
			Type: schema.NewListType(schema.NewNonNullType(EnumValueType)),
			Cost: schema.FieldResolverCost(0),
			Arguments: map[string]*schema.InputValueDefinition{
				"includeDeprecated": {
					Type:         schema.BooleanType,
					DefaultValue: false,
				},
			},
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				if t, ok := ctx.Object.(*schema.EnumType); ok {
					includeDeprecated := ctx.Arguments["includeDeprecated"].(bool)
					ret := []enumValue{}
					for name, def := range t.Values {
						if def.DeprecationReason == "" || includeDeprecated {
							ret = append(ret, enumValue{
								Name:       name,
								Definition: def,
							})
						}
					}
					return ret, nil
				}
				return nil, nil
			},
		},
		"inputFields": {
			Type: schema.NewListType(schema.NewNonNullType(InputValueType)),
			Cost: schema.FieldResolverCost(0),
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				if t, ok := ctx.Object.(*schema.InputObjectType); ok {
					return inputValues(t.Fields)
				}
				return nil, nil
			},
		},
		"ofType": {
			Type: TypeType,
			Cost: schema.FieldResolverCost(0),
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				switch t := ctx.Object.(type) {
				case *schema.ListType:
					return t.Type, nil
				case *schema.NonNullType:
					return t.Type, nil
				default:
					return nil, nil
				}
			},
		},
	}
}

var DirectiveLocationType = &schema.EnumType{
	Name: "__DirectiveLocation",
	Values: map[string]*schema.EnumValueDefinition{
		"QUERY": {
			Value: schema.DirectiveLocationQuery,
		},
		"MUTATION": {
			Value: schema.DirectiveLocationMutation,
		},
		"SUBSCRIPTION": {
			Value: schema.DirectiveLocationSubscription,
		},
		"FIELD": {
			Value: schema.DirectiveLocationField,
		},
		"FRAGMENT_DEFINITION": {
			Value: schema.DirectiveLocationFragmentDefinition,
		},
		"FRAGMENT_SPREAD": {
			Value: schema.DirectiveLocationFragmentSpread,
		},
		"INLINE_FRAGMENT": {
			Value: schema.DirectiveLocationInlineFragment,
		},
		"SCHEMA": {
			Value: schema.DirectiveLocationSchema,
		},
		"SCALAR": {
			Value: schema.DirectiveLocationScalar,
		},
		"OBJECT": {
			Value: schema.DirectiveLocationObject,
		},
		"FIELD_DEFINITION": {
			Value: schema.DirectiveLocationFieldDefinition,
		},
		"ARGUMENT_DEFINITION": {
			Value: schema.DirectiveLocationArgumentDefinition,
		},
		"INTERFACE": {
			Value: schema.DirectiveLocationInterface,
		},
		"UNION": {
			Value: schema.DirectiveLocationUnion,
		},
		"ENUM": {
			Value: schema.DirectiveLocationEnum,
		},
		"ENUM_VALUE": {
			Value: schema.DirectiveLocationEnumValue,
		},
		"INPUT_OBJECT": {
			Value: schema.DirectiveLocationInputObject,
		},
		"INPUT_FIELD_DEFINITION": {
			Value: schema.DirectiveLocationInputFieldDefinition,
		},
	},
}

var DirectiveType = &schema.ObjectType{
	Name: "__Directive",
	Fields: map[string]*schema.FieldDefinition{
		"name": {
			Type: schema.NewNonNullType(schema.StringType),
			Cost: schema.FieldResolverCost(0),
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				return ctx.Object.(directive).Name, nil
			},
		},
		"description": {
			Type: schema.StringType,
			Cost: schema.FieldResolverCost(0),
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				return nullableString(ctx.Object.(directive).Definition.Description)
			},
		},
		"locations": {
			Type: schema.NewNonNullType(schema.NewListType(schema.NewNonNullType(DirectiveLocationType))),
			Cost: schema.FieldResolverCost(0),
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				return ctx.Object.(directive).Definition.Locations, nil
			},
		},
		"args": {
			Type: schema.NewNonNullType(schema.NewListType(schema.NewNonNullType(InputValueType))),
			Cost: schema.FieldResolverCost(0),
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				return inputValues(ctx.Object.(directive).Definition.Arguments)
			},
		},
	},
}

type field struct {
	Name       string
	Definition *schema.FieldDefinition
}

var FieldType = &schema.ObjectType{
	Name: "__Field",
	Fields: map[string]*schema.FieldDefinition{
		"name": {
			Type: schema.NewNonNullType(schema.StringType),
			Cost: schema.FieldResolverCost(0),
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				return ctx.Object.(field).Name, nil
			},
		},
		"description": {
			Type: schema.StringType,
			Cost: schema.FieldResolverCost(0),
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				return nullableString(ctx.Object.(field).Definition.Description)
			},
		},
		"args": {
			Type: schema.NewNonNullType(schema.NewListType(schema.NewNonNullType(InputValueType))),
			Cost: schema.FieldResolverCost(0),
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				return inputValues(ctx.Object.(field).Definition.Arguments)
			},
		},
		"type": {
			Type: schema.NewNonNullType(TypeType),
			Cost: schema.FieldResolverCost(0),
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				return ctx.Object.(field).Definition.Type, nil
			},
		},
		"isDeprecated": {
			Type: schema.NewNonNullType(schema.BooleanType),
			Cost: schema.FieldResolverCost(0),
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				return ctx.Object.(field).Definition.DeprecationReason != "", nil
			},
		},
		"deprecationReason": {
			Type: schema.StringType,
			Cost: schema.FieldResolverCost(0),
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				return nullableString(ctx.Object.(field).Definition.DeprecationReason)
			},
		},
	},
}

type enumValue struct {
	Name       string
	Definition *schema.EnumValueDefinition
}

var EnumValueType = &schema.ObjectType{
	Name: "__EnumValue",
	Fields: map[string]*schema.FieldDefinition{
		"name": {
			Type: schema.NewNonNullType(schema.StringType),
			Cost: schema.FieldResolverCost(0),
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				return ctx.Object.(enumValue).Name, nil
			},
		},
		"description": {
			Type: schema.StringType,
			Cost: schema.FieldResolverCost(0),
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				return nullableString(ctx.Object.(enumValue).Definition.Description)
			},
		},
		"isDeprecated": {
			Type: schema.NewNonNullType(schema.BooleanType),
			Cost: schema.FieldResolverCost(0),
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				return ctx.Object.(enumValue).Definition.DeprecationReason != "", nil
			},
		},
		"deprecationReason": {
			Type: schema.StringType,
			Cost: schema.FieldResolverCost(0),
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				return nullableString(ctx.Object.(enumValue).Definition.DeprecationReason)
			},
		},
	},
}

type inputValue struct {
	Name       string
	Definition *schema.InputValueDefinition
}

var InputValueType = &schema.ObjectType{
	Name: "__InputValue",
	Fields: map[string]*schema.FieldDefinition{
		"name": {
			Type: schema.NewNonNullType(schema.StringType),
			Cost: schema.FieldResolverCost(0),
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				return ctx.Object.(inputValue).Name, nil
			},
		},
		"description": {
			Type: schema.StringType,
			Cost: schema.FieldResolverCost(0),
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				return nullableString(ctx.Object.(inputValue).Definition.Description)
			},
		},
		"type": {
			Type: schema.NewNonNullType(TypeType),
			Cost: schema.FieldResolverCost(0),
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				return ctx.Object.(inputValue).Definition.Type, nil
			},
		},
		"defaultValue": {
			Type: schema.StringType,
			Cost: schema.FieldResolverCost(0),
			Resolve: func(ctx schema.FieldContext) (interface{}, error) {
				def := ctx.Object.(inputValue).Definition
				if v := def.DefaultValue; v != nil {
					return marshalValue(def.Type, v)
				}
				return nil, nil
			},
		},
	},
}
