package schema

import (
	"fmt"
	"regexp"
	"strings"
)

type Schema struct {
	directiveDefinitions     map[string]*DirectiveDefinition
	namedTypes               map[string]NamedType
	interfaceImplementations map[string][]*ObjectType

	query        *ObjectType
	mutation     *ObjectType
	subscription *ObjectType
}

func (s *Schema) QueryType() *ObjectType {
	return s.query
}

func (s *Schema) MutationType() *ObjectType {
	return s.query
}

func (s *Schema) SubscriptionType() *ObjectType {
	return s.subscription
}

func (s *Schema) DirectiveDefinition(name string) *DirectiveDefinition {
	return s.directiveDefinitions[name]
}

func (s *Schema) NamedType(name string) NamedType {
	return s.namedTypes[name]
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
		directiveDefinitions:     def.DirectiveDefinitions,
		namedTypes:               map[string]NamedType{},
		interfaceImplementations: map[string][]*ObjectType{},
		query:                    def.Query,
		mutation:                 def.Mutation,
		subscription:             def.Subscription,
	}

	if schema.query == nil {
		return nil, fmt.Errorf("schemas must define the query operation")
	}

	for name := range def.DirectiveDefinitions {
		if !isName(name) || strings.HasPrefix(name, "__") {
			return nil, fmt.Errorf("illegal directive name: %v", name)
		}
	}

	Inspect(def, func(node interface{}) bool {
		if err != nil {
			return false
		}

		if namedType, ok := node.(NamedType); ok {
			if name := namedType.NamedType(); !isName(name) || strings.HasPrefix(name, "__") {
				err = fmt.Errorf("illegal type name: %v", name)
			} else if existing, ok := schema.namedTypes[name]; ok && existing != namedType {
				err = fmt.Errorf("multiple definitions for named type: %v", name)
			} else if builtin, ok := builtins[name]; ok && namedType != builtin {
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
	Directives           []*Directive
	DirectiveDefinitions map[string]*DirectiveDefinition

	Query        *ObjectType
	Mutation     *ObjectType
	Subscription *ObjectType
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
	NamedType() string
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
