package introspection

import (
	"fmt"

	"github.com/ccbrown/api-fu/graphql/schema"
)

type SchemaData struct {
	QueryType        TypeData
	MutationType     *TypeData
	SubscriptionType *TypeData
	Types            []TypeData
	Directives       []DirectiveData
}

// Gets a schema definition for the given schema data. This is not a lossless transformation, and
// the definition will not be usable for a server as-is, but it can be used for example to validate
// a query against another server's GraphQL schema.
func (d *SchemaData) GetSchemaDefinition() (*schema.SchemaDefinition, error) {
	ret := &schema.SchemaDefinition{
		Directives: map[string]*schema.DirectiveDefinition{},
	}

	types := map[string]schema.NamedType{}

	for _, t := range d.Types {
		if builtin, ok := schema.BuiltInTypes[t.Name]; ok {
			types[t.Name] = builtin
			continue
		}

		switch t.Kind {
		case "SCALAR":
			types[t.Name] = &schema.ScalarType{}
		case "OBJECT":
			types[t.Name] = &schema.ObjectType{}
		case "INTERFACE":
			types[t.Name] = &schema.InterfaceType{}
		case "UNION":
			types[t.Name] = &schema.UnionType{}
		case "ENUM":
			types[t.Name] = &schema.EnumType{}
		case "INPUT_OBJECT":
			types[t.Name] = &schema.InputObjectType{}
		default:
			return nil, fmt.Errorf("unsupported type kind in types list: %v", t.Kind)
		}
	}

	if t, err := d.QueryType.getType(types); err != nil {
		return nil, err
	} else if obj, ok := t.(*schema.ObjectType); !ok {
		return nil, fmt.Errorf("query type is not an object")
	} else {
		ret.Query = obj
	}

	if d.MutationType != nil {
		if t, err := d.MutationType.getType(types); err != nil {
			return nil, err
		} else if obj, ok := t.(*schema.ObjectType); !ok {
			return nil, fmt.Errorf("mutation type is not an object")
		} else {
			ret.Mutation = obj
		}
	}

	if d.SubscriptionType != nil {
		if t, err := d.SubscriptionType.getType(types); err != nil {
			return nil, err
		} else if obj, ok := t.(*schema.ObjectType); !ok {
			return nil, fmt.Errorf("subcription type is not an object")
		} else {
			ret.Subscription = obj
		}
	}

	for _, t := range d.Types {
		if _, ok := schema.BuiltInTypes[t.Name]; ok {
			continue
		}

		switch t.Kind {
		case "SCALAR":
			def := types[t.Name].(*schema.ScalarType)
			def.Name = t.Name
			def.Description = t.Description
		case "OBJECT":
			def := types[t.Name].(*schema.ObjectType)
			def.Name = t.Name
			def.Description = t.Description
			def.Fields = map[string]*schema.FieldDefinition{}
			for _, field := range t.Fields {
				if fieldDef, err := field.getFieldDefinition(types); err != nil {
					return nil, err
				} else {
					def.Fields[field.Name] = fieldDef
				}
			}
			for _, t := range t.Interfaces {
				if iface, err := t.getType(types); err != nil {
					return nil, err
				} else if iface, ok := iface.(*schema.InterfaceType); !ok {
					return nil, fmt.Errorf("type is not an interface: %s", t.Name)
				} else {
					def.ImplementedInterfaces = append(def.ImplementedInterfaces, iface)
				}
			}
			def.IsTypeOf = func(v interface{}) bool {
				return false
			}
		case "INTERFACE":
			def := types[t.Name].(*schema.InterfaceType)
			def.Name = t.Name
			def.Description = t.Description
			def.Fields = map[string]*schema.FieldDefinition{}
			for _, field := range t.Fields {
				if fieldDef, err := field.getFieldDefinition(types); err != nil {
					return nil, err
				} else {
					def.Fields[field.Name] = fieldDef
				}
			}
		case "UNION":
			def := types[t.Name].(*schema.UnionType)
			def.Name = t.Name
			def.Description = t.Description
			for _, t := range t.PossibleTypes {
				if obj, err := t.getType(types); err != nil {
					return nil, err
				} else if obj, ok := obj.(*schema.ObjectType); !ok {
					return nil, fmt.Errorf("type is not an object: %s", t.Name)
				} else {
					def.MemberTypes = append(def.MemberTypes, obj)
				}
			}
		case "ENUM":
			def := types[t.Name].(*schema.EnumType)
			def.Name = t.Name
			def.Description = t.Description
			def.Values = map[string]*schema.EnumValueDefinition{}
			for _, value := range t.EnumValues {
				if valueDef, err := value.getEnumValueDefinition(types); err != nil {
					return nil, err
				} else {
					def.Values[value.Name] = valueDef
				}
			}
		case "INPUT_OBJECT":
			def := types[t.Name].(*schema.InputObjectType)
			def.Name = t.Name
			def.Description = t.Description
			def.Fields = map[string]*schema.InputValueDefinition{}
			for _, field := range t.InputFields {
				if fieldDef, err := field.getInputValueDefinition(types); err != nil {
					return nil, err
				} else {
					def.Fields[field.Name] = fieldDef
				}
			}
		}
	}

	for _, dir := range d.Directives {
		if def, err := dir.getDirectiveDefinition(types); err != nil {
			return nil, err
		} else {
			ret.Directives[dir.Name] = def
		}
	}

	return ret, nil
}

type DirectiveData struct {
	Name        string
	Description string
	Locations   []string
	Args        []InputValueData
}

var directiveLocations = map[string]schema.DirectiveLocation{
	"QUERY":                  schema.DirectiveLocationQuery,
	"MUTATION":               schema.DirectiveLocationMutation,
	"SUBSCRIPTION":           schema.DirectiveLocationSubscription,
	"FIELD":                  schema.DirectiveLocationField,
	"FRAGMENT_DEFINITION":    schema.DirectiveLocationFragmentDefinition,
	"FRAGMENT_SPREAD":        schema.DirectiveLocationFragmentSpread,
	"INLINE_FRAGMENT":        schema.DirectiveLocationInlineFragment,
	"SCHEMA":                 schema.DirectiveLocationSchema,
	"SCALAR":                 schema.DirectiveLocationScalar,
	"OBJECT":                 schema.DirectiveLocationObject,
	"FIELD_DEFINITION":       schema.DirectiveLocationFieldDefinition,
	"ARGUMENT_DEFINITION":    schema.DirectiveLocationArgumentDefinition,
	"INTERFACE":              schema.DirectiveLocationInterface,
	"UNION":                  schema.DirectiveLocationUnion,
	"ENUM":                   schema.DirectiveLocationEnum,
	"ENUM_VALUE":             schema.DirectiveLocationEnumValue,
	"INPUT_OBJECT":           schema.DirectiveLocationInputObject,
	"INPUT_FIELD_DEFINITION": schema.DirectiveLocationInputFieldDefinition,
}

func (d DirectiveData) getDirectiveDefinition(types map[string]schema.NamedType) (*schema.DirectiveDefinition, error) {
	ret := &schema.DirectiveDefinition{
		Description: d.Description,
		Arguments:   map[string]*schema.InputValueDefinition{},
	}
	for _, l := range d.Locations {
		if def, ok := directiveLocations[l]; ok {
			ret.Locations = append(ret.Locations, def)
		} else {
			return nil, fmt.Errorf("unsupported directive location: %v", l)
		}
	}
	for _, arg := range d.Args {
		if def, err := arg.getInputValueDefinition(types); err != nil {
			return nil, err
		} else {
			ret.Arguments[arg.Name] = def
		}
	}
	return ret, nil
}

type TypeData struct {
	Kind          string
	Name          string
	Description   string
	Fields        []FieldData
	InputFields   []InputValueData
	Interfaces    []TypeData
	EnumValues    []EnumValueData
	PossibleTypes []TypeData
	OfType        *TypeData
}

func (d TypeData) getType(types map[string]schema.NamedType) (schema.Type, error) {
	switch d.Kind {
	case "LIST":
		if d.OfType == nil {
			return nil, fmt.Errorf("null ofType for list type")
		} else if ofType, err := d.OfType.getType(types); err != nil {
			return nil, err
		} else {
			return schema.NewListType(ofType), nil
		}
	case "NON_NULL":
		if d.OfType == nil {
			return nil, fmt.Errorf("null ofType for non-null type")
		} else if ofType, err := d.OfType.getType(types); err != nil {
			return nil, err
		} else {
			return schema.NewNonNullType(ofType), nil
		}
	default:
		if t := types[d.Name]; t != nil {
			return t, nil
		}
	}
	return nil, fmt.Errorf("type not found: %v", d.Name)
}

type FieldData struct {
	Name              string
	Description       string
	Args              []InputValueData
	Type              TypeData
	IsDeprecated      bool
	DeprecationReason string
}

func (d FieldData) getFieldDefinition(types map[string]schema.NamedType) (*schema.FieldDefinition, error) {
	t, err := d.Type.getType(types)
	if err != nil {
		return nil, err
	}
	ret := &schema.FieldDefinition{
		Description:       d.Description,
		DeprecationReason: d.DeprecationReason,
		Type:              t,
		Arguments:         map[string]*schema.InputValueDefinition{},
	}
	for _, arg := range d.Args {
		if def, err := arg.getInputValueDefinition(types); err != nil {
			return nil, err
		} else {
			ret.Arguments[arg.Name] = def
		}
	}
	return ret, nil
}

type InputValueData struct {
	Name        string
	Description string
	Type        TypeData
}

func (d InputValueData) getInputValueDefinition(types map[string]schema.NamedType) (*schema.InputValueDefinition, error) {
	t, err := d.Type.getType(types)
	if err != nil {
		return nil, err
	}
	return &schema.InputValueDefinition{
		Description: d.Description,
		Type:        t,
	}, nil
}

type EnumValueData struct {
	Name              string
	Description       string
	IsDeprecated      bool
	DeprecationReason string
}

func (d EnumValueData) getEnumValueDefinition(types map[string]schema.NamedType) (*schema.EnumValueDefinition, error) {
	return &schema.EnumValueDefinition{
		Description:       d.Description,
		DeprecationReason: d.DeprecationReason,
	}, nil
}
