package schema

import "fmt"

type NonNullType struct {
	Type Type
}

func NewNonNullType(t Type) *NonNullType {
	return &NonNullType{
		Type: t,
	}
}

func (t *NonNullType) String() string {
	return t.Type.String() + "!"
}

func (t *NonNullType) IsInputType() bool {
	return t.Type.IsInputType()
}

func (t *NonNullType) IsOutputType() bool {
	return t.Type.IsOutputType()
}

func (t *NonNullType) IsSubTypeOf(other Type) bool {
	return t.IsSameType(other) || t.Type.IsSubTypeOf(other)
}

func (t *NonNullType) IsSameType(other Type) bool {
	if nn, ok := other.(*NonNullType); ok {
		return t.Type.IsSameType(nn.Type)
	}
	return false
}

func (t *NonNullType) Unwrap() Type {
	return t.Type
}

func (t *NonNullType) shallowValidate() error {
	if IsNonNullType(t.Type) {
		return fmt.Errorf("non-null types cannot wrap other non-null types")
	}
	return nil
}

func IsNonNullType(t Type) bool {
	_, ok := t.(*NonNullType)
	return ok
}

func NullableType(t Type) Type {
	for {
		if nnt, ok := t.(*NonNullType); ok {
			t = nnt.Unwrap()
		} else {
			break
		}
	}
	return t
}
