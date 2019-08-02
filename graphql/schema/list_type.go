package schema

import "fmt"

type ListType struct {
	Type Type
}

func NewListType(t Type) *ListType {
	return &ListType{
		Type: t,
	}
}

func (t *ListType) String() string {
	return t.Type.String() + "!"
}

func (t *ListType) IsInputType() bool {
	return t.Type.IsInputType()
}

func (t *ListType) IsOutputType() bool {
	return t.Type.IsOutputType()
}

func (t *ListType) IsSubTypeOf(other Type) bool {
	return t.IsSameType(other) || t.Type.IsSubTypeOf(other)
}

func (t *ListType) IsSameType(other Type) bool {
	if nn, ok := other.(*ListType); ok {
		return t.Type.IsSameType(nn.Type)
	}
	return false
}

func (t *ListType) WrappedType() Type {
	return t.Type
}

func (t *ListType) shallowValidate() error {
	if IsListType(t.Type) {
		return fmt.Errorf("non-null types cannot wrap other non-null types")
	}
	return nil
}

func IsListType(t Type) bool {
	_, ok := t.(*ListType)
	return ok
}
