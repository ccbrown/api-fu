package schema

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ccbrown/api-fu/graphql/ast"
)

type Schema struct {
	directives               map[string]*DirectiveDefinition
	namedTypes               map[string]NamedType
	interfaceImplementations map[string][]*ObjectType

	queryType        *ObjectType
	mutationType     *ObjectType
	subscriptionType *ObjectType
}

func (s *Schema) QueryType() *ObjectType {
	return s.queryType
}

func (s *Schema) MutationType() *ObjectType {
	return s.mutationType
}

func (s *Schema) SubscriptionType() *ObjectType {
	return s.subscriptionType
}

func (s *Schema) Directives() map[string]*DirectiveDefinition {
	return s.directives
}

func (s *Schema) NamedTypes() map[string]NamedType {
	return s.namedTypes
}

func (s *Schema) InterfaceImplementations(name string) []*ObjectType {
	return s.interfaceImplementations[name]
}

var nameRegex = regexp.MustCompile(`^[_A-Za-z][_0-9A-Za-z]*$`)

func isName(s string) bool {
	return nameRegex.MatchString(s)
}

func New(def *SchemaDefinition) (*Schema, error) {
	var err error
	schema := &Schema{
		directives:               def.Directives,
		namedTypes:               map[string]NamedType{},
		interfaceImplementations: map[string][]*ObjectType{},
		queryType:                def.Query,
		mutationType:             def.Mutation,
		subscriptionType:         def.Subscription,
	}

	if schema.queryType == nil {
		return nil, fmt.Errorf("schemas must define the query operation")
	}

	for name := range def.Directives {
		if !isName(name) || strings.HasPrefix(name, "__") {
			return nil, fmt.Errorf("illegal directive name: %v", name)
		}
	}

	Inspect(def, func(node interface{}) bool {
		if err != nil {
			return false
		}

		if namedType, ok := node.(NamedType); ok {
			if name := namedType.TypeName(); !isName(name) || strings.HasPrefix(name, "__") {
				err = fmt.Errorf("illegal type name: %v", name)
			} else if existing, ok := schema.namedTypes[name]; ok && existing != namedType {
				err = fmt.Errorf("multiple definitions for named type: %v", name)
			} else if builtin, ok := builtinTypes[name]; ok && namedType != builtin {
				err = fmt.Errorf("%v builtin may not be overridden", name)
			} else if existing != nil {
				// already visited
				return false
			} else {
				schema.namedTypes[name] = namedType
			}
		}

		if obj, ok := node.(*ObjectType); ok {
			for _, iface := range obj.ImplementedInterfaces {
				schema.interfaceImplementations[iface.Name] = append(schema.interfaceImplementations[iface.Name], obj)
			}
		}

		if err == nil {
			if n, ok := node.(interface {
				shallowValidate() error
			}); ok {
				err = n.shallowValidate()
			}
		}

		return err == nil
	})

	if err != nil {
		return nil, err
	}
	return schema, nil
}

type SchemaDefinition struct {
	// Directives to define within the schema. For example, you might want to add IncludeDirective
	// and SkipDirective here.
	Directives map[string]*DirectiveDefinition

	Query        *ObjectType
	Mutation     *ObjectType
	Subscription *ObjectType

	// AdditionalTypes is used to add otherwise unreferenced types to the schema.
	AdditionalTypes []NamedType
}

type Argument struct {
	Name  string
	Value interface{}
}

type Type interface {
	String() string
	IsInputType() bool
	IsOutputType() bool
	IsSubTypeOf(Type) bool
	IsSameType(Type) bool
}

type NamedType interface {
	Type
	TypeName() string
}

type WrappedType interface {
	Type
	Unwrap() Type
}

func UnwrappedType(t Type) NamedType {
	for {
		if wrapped, ok := t.(WrappedType); ok {
			t = wrapped.Unwrap()
		} else {
			break
		}
	}
	if t != nil {
		return t.(NamedType)
	}
	return nil
}

func CoerceVariableValue(value interface{}, t Type) (interface{}, error) {
	return coerceVariableValue(value, t, true)
}

func coerceVariableValue(value interface{}, t Type, allowItemToListCoercion bool) (interface{}, error) {
	if value == nil {
		if IsNonNullType(t) {
			return nil, fmt.Errorf("a value is required")
		}
		return nil, nil
	}

	switch t := t.(type) {
	case *ScalarType:
		return t.CoerceVariableValue(value)
	case *EnumType:
		return t.CoerceVariableValue(value)
	case *InputObjectType:
		return t.CoerceVariableValue(value)
	case *ListType:
		return t.coerceVariableValue(value, allowItemToListCoercion)
	case *NonNullType:
		return CoerceVariableValue(value, t.Type)
	default:
		panic("unexpected variable coercion type")
	}
}

func CoerceLiteral(from ast.Value, to Type, variableValues map[string]interface{}) (interface{}, error) {
	return coerceLiteral(from, to, variableValues, true)
}

func coerceLiteral(from ast.Value, to Type, variableValues map[string]interface{}, allowItemToListCoercion bool) (interface{}, error) {
	if ast.IsNullValue(from) {
		if IsNonNullType(to) {
			return nil, fmt.Errorf("cannot coerce null to non-null type")
		}
		return nil, nil
	} else if variable, ok := from.(*ast.Variable); ok {
		if value, ok := variableValues[variable.Name.Name]; ok {
			return value, nil
		}
	}

	switch to := to.(type) {
	case *ScalarType:
		if v := to.LiteralCoercion(from); v != nil {
			return v, nil
		}
		return nil, fmt.Errorf("cannot coerce to %v", to)
	case *ListType:
		return to.coerceLiteral(from, variableValues, allowItemToListCoercion)
	case *InputObjectType:
		if v, ok := from.(*ast.ObjectValue); ok {
			return to.CoerceLiteral(v, variableValues)
		}
		return nil, fmt.Errorf("cannot coerce to %v", to)
	case *EnumType:
		return to.CoerceLiteral(from)
	case *NonNullType:
		return CoerceLiteral(from, to.Type, variableValues)
	}

	panic("unsupported literal coercion type")
}
