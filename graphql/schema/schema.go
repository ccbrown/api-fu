package schema

import (
	"fmt"
	"regexp"
	"strings"
)

type Schema struct {
	directives map[string]*DirectiveDefinition
	namedTypes map[string]NamedType

	query        *ObjectDefinition
	mutation     *ObjectDefinition
	subscription *ObjectDefinition
}

var nameRegex = regexp.MustCompile(`^[_A-Za-z][_0-9A-Za-z]*$`)

func isName(s string) bool {
	return nameRegex.MatchString(s)
}

func referencesDirective(node interface{}, directive *DirectiveDefinition) bool {
	visited := map[interface{}]struct{}{}
	foundReference := false

	Inspect(node, func(node interface{}) bool {
		if _, ok := visited[node]; ok {
			return false
		}
		visited[node] = struct{}{}
		if node == directive {
			foundReference = true
		}
		return !foundReference
	})

	return foundReference
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
		return nil, fmt.Errorf("schemas must support query operations")
	}

	Inspect(def, func(node interface{}) bool {
		if nonNull, ok := node.(*NonNullType); ok {
			if isNonNull(nonNull.Type) {
				err = fmt.Errorf("non-null types cannot wrap other non-null types")
			}
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
			} else {
				schema.namedTypes[name] = namedType
			}
		}

		if err != nil {
			return false
		}

		switch n := node.(type) {
		case *DirectiveDefinition:
			if name := n.Name; !isName(name) || strings.HasPrefix(name, "__") {
				err = fmt.Errorf("illegal directive name: %v", name)
			} else if existing, ok := schema.directives[name]; ok && existing != n {
				err = fmt.Errorf("multiple definitions for directive: %v", name)
			} else if !ok {
				schema.directives[name] = n

				for name, arg := range n.Arguments {
					if !isName(name) || strings.HasPrefix(name, "__") {
						err = fmt.Errorf("illegal directive argument name: %v", name)
						return false
					} else if referencesDirective(arg, n) {
						err = fmt.Errorf("directive is self-referencing via %v argument", name)
						return false
					}
				}
			}
		case *InterfaceDefinition:
			if len(n.Fields) == 0 {
				err = fmt.Errorf("%v must have at least one field", n.Name)
			} else {
				for name := range n.Fields {
					if !isName(name) || strings.HasPrefix(name, "__") {
						err = fmt.Errorf("illegal field name: %v", name)
						break
					}
				}
			}
		case *ObjectDefinition:
			if len(n.Fields) == 0 {
				err = fmt.Errorf("%v must have at least one field", n.Name)
			} else {
				for name := range n.Fields {
					if !isName(name) || strings.HasPrefix(name, "__") {
						err = fmt.Errorf("illegal field name: %v", name)
						break
					}
				}
			}
		case *InputValueDefinition:
			if n.Type == nil {
				err = fmt.Errorf("input value is missing type")
			} else if !n.Type.IsInputType() {
				err = fmt.Errorf("%v cannot be used as an input value type", n.Type)
			}
		case *FieldDefinition:
			if n.Type == nil {
				err = fmt.Errorf("field is missing type")
			} else if !n.Type.IsOutputType() {
				err = fmt.Errorf("%v cannot be used as a field type", n.Type)
			} else {
				for name := range n.Arguments {
					if !isName(name) || strings.HasPrefix(name, "__") {
						err = fmt.Errorf("illegal field argument name: %v", name)
						break
					}
				}
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

	Query        *ObjectDefinition
	Mutation     *ObjectDefinition
	Subscription *ObjectDefinition
}

type Argument struct {
	Name  string
	Value interface{}
}

type InputValueDefinition struct {
	Description  string
	Type         Type
	DefaultValue interface{}
	Directives   []*Directive
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

type FieldDefinition struct {
	Description string
	Arguments   map[string]*InputValueDefinition
	Type        Type
	Directives  []*Directive
}
