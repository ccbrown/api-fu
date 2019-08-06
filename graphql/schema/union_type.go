package schema

import "fmt"

type UnionType struct {
	Name        string
	Description string
	Directives  []*Directive
	MemberTypes []NamedType
	ObjectType  func(object interface{}) *ObjectType
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

func (d *UnionType) NamedType() string {
	return d.Name
}

func (d *UnionType) shallowValidate() error {
	if len(d.MemberTypes) == 0 {
		return fmt.Errorf("%v must have at least one member type", d.Name)
	} else {
		objNames := map[string]struct{}{}
		for _, member := range d.MemberTypes {
			if obj, ok := member.(*ObjectType); ok {
				if _, ok := objNames[obj.Name]; ok {
					return fmt.Errorf("union member types must be unique")
				}
				objNames[obj.Name] = struct{}{}
			} else {
				return fmt.Errorf("union member types must be objects")
			}
		}
	}
	return nil
}
