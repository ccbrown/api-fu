package schema

import "fmt"

type UnionType struct {
	Name        string
	Description string
	Directives  []*Directive
	MemberTypes []*ObjectType
}

func (d *UnionType) String() string {
	return d.Name
}

func (d *UnionType) IsInputType() bool {
	return false
}

func (d *UnionType) IsOutputType() bool {
	return true
}

func (d *UnionType) IsSubTypeOf(other Type) bool {
	return d.IsSameType(other)
}

func (d *UnionType) IsSameType(other Type) bool {
	return d == other
}

func (d *UnionType) TypeName() string {
	return d.Name
}

func (d *UnionType) shallowValidate() error {
	if len(d.MemberTypes) == 0 {
		return fmt.Errorf("%v must have at least one member type", d.Name)
	}
	objNames := map[string]struct{}{}
	for _, member := range d.MemberTypes {
		if _, ok := objNames[member.Name]; ok {
			return fmt.Errorf("union member types must be unique")
		}
		if member.IsTypeOf == nil {
			return fmt.Errorf("union member types must define IsTypeOf")
		}
		objNames[member.Name] = struct{}{}
	}
	return nil
}
