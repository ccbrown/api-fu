package schema

type InterfaceDefinition struct {
	Name        string
	Description string
	Directives  []*Directive
	Fields      map[string]*FieldDefinition
}

func (d *InterfaceDefinition) String() string {
	return d.Name
}

func (d *InterfaceDefinition) IsInputType() bool {
	return false
}

func (d *InterfaceDefinition) IsOutputType() bool {
	return true
}

func (d *InterfaceDefinition) IsSubTypeOf(other Type) bool {
	return d.IsSameType(other)
}

func (d *InterfaceDefinition) IsSameType(other Type) bool {
	return d == other
}

func (d *InterfaceDefinition) NamedType() string {
	return d.Name
}
