package schema

import "fmt"

type ObjectDefinition struct {
	Name                  string
	Description           string
	ImplementedInterfaces []*InterfaceDefinition
	Directives            []*Directive
	Fields                map[string]*FieldDefinition
}

func (d *ObjectDefinition) String() string {
	return d.Name
}

func (d *ObjectDefinition) IsInputType() bool {
	return false
}

func (d *ObjectDefinition) IsOutputType() bool {
	return true
}

func (d *ObjectDefinition) IsSubTypeOf(other Type) bool {
	if d.IsSameType(other) {
		return true
	}
	for _, iface := range d.ImplementedInterfaces {
		if iface == other {
			return true
		}
	}
	return false
}

func (d *ObjectDefinition) IsSameType(other Type) bool {
	return d == other
}

func (d *ObjectDefinition) NamedType() string {
	return d.Name
}

func (d *ObjectDefinition) SatisfyInterface(iface *InterfaceDefinition) error {
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
			if _, ok := ifaceField.Arguments[argName]; !ok && isNonNull(arg.Type) {
				return fmt.Errorf("object's %v field %v argument cannot be non-null", name, argName)
			}
		}
	}
	return nil
}
