package schema

import (
	"fmt"
	"strings"
)

type ObjectType struct {
	Name                  string
	Description           string
	ImplementedInterfaces []*InterfaceType
	Directives            []*Directive
	Fields                map[string]*FieldDefinition
}

func (d *ObjectType) String() string {
	return d.Name
}

func (d *ObjectType) IsInputType() bool {
	return false
}

func (d *ObjectType) IsOutputType() bool {
	return true
}

func (d *ObjectType) IsSubTypeOf(other Type) bool {
	if d.IsSameType(other) {
		return true
	} else if union, ok := other.(*UnionType); ok {
		for _, member := range union.MemberTypes {
			if d.IsSameType(member) {
				return true
			}
		}
	} else {
		for _, iface := range d.ImplementedInterfaces {
			if iface.IsSameType(other) {
				return true
			}
		}
	}
	return false
}

func (d *ObjectType) IsSameType(other Type) bool {
	return d == other
}

func (d *ObjectType) NamedType() string {
	return d.Name
}

func (d *ObjectType) SatisfyInterface(iface *InterfaceType) error {
	for name, ifaceField := range iface.Fields {
		field, ok := d.Fields[name]
		if !ok {
			return fmt.Errorf("object is missing field named %v", name)
		} else if !field.Type.IsSubTypeOf(ifaceField.Type) {
			return fmt.Errorf("object's %v field is not a subtype of the corresponding interface field", name)
		}
		for argName, ifaceArg := range ifaceField.Arguments {
			arg, ok := field.Arguments[argName]
			if !ok {
				return fmt.Errorf("object's %v field is missing argument named %v", name, argName)
			} else if !arg.Type.IsSameType(ifaceArg.Type) {
				return fmt.Errorf("object's %v field %v argument is not the same type as the corresponding interface argument", name, argName)
			}
		}
		for argName, arg := range field.Arguments {
			if _, ok := ifaceField.Arguments[argName]; !ok && IsNonNullType(arg.Type) {
				return fmt.Errorf("object's %v field %v argument cannot be non-null", name, argName)
			}
		}
	}
	return nil
}

func (d *ObjectType) shallowValidate() error {
	if len(d.Fields) == 0 {
		return fmt.Errorf("%v must have at least one field", d.Name)
	} else {
		for name, field := range d.Fields {
			if !isName(name) || strings.HasPrefix(name, "__") {
				return fmt.Errorf("illegal field name: %v", name)
			} else if !field.Type.IsOutputType() {
				return fmt.Errorf("%v field must be an output type", name)
			}
		}
	}
	return nil
}
