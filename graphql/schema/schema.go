package schema

import (
	"fmt"
	"regexp"
	"strings"
)

type Schema struct {
	directives map[string]*DirectiveDefinition
	namedTypes map[string]NamedType

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

func (s *Schema) NamedType(name string) NamedType {
	return s.namedTypes[name]
}

var nameRegex = regexp.MustCompile(`^[_A-Za-z][_0-9A-Za-z]*$`)

func isName(s string) bool {
	return nameRegex.MatchString(s)
}

func New(def *SchemaDefinition) (*Schema, error) {
	var err error
	schema := &Schema{
		directives:   map[string]*DirectiveDefinition{},
		namedTypes:   map[string]NamedType{},
		query:        def.Query,
		mutation:     def.Mutation,
		subscription: def.Subscription,
	}

	if schema.query == nil {
		return nil, fmt.Errorf("schemas must define the query operation")
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
			} else if ok {
				// already visited
				return false
			} else {
				schema.namedTypes[name] = namedType
			}
		}

		if d, ok := node.(*DirectiveDefinition); ok {
			if existing, ok := schema.directives[d.Name]; ok && existing != d {
				err = fmt.Errorf("multiple definitions for directive: %v", d.Name)
			} else if ok {
				// already visited
				return false
			} else {
				schema.directives[d.Name] = d
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
	Directives []*Directive

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

func UnwrapType(t Type) Type {
	for {
		if wrapped, ok := t.(WrappedType); ok {
			t = wrapped.Unwrap()
		} else {
			break
		}
	}
	return t
}
