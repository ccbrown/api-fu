package schema

type NonNullType struct {
	Type Type
}

func NonNull(t Type) *NonNullType {
	return &NonNullType{
		Type: t,
	}
}

func (d *NonNullType) String() string {
	return d.Type.String() + "!"
}

func (d *NonNullType) IsInputType() bool {
	return d.Type.IsInputType()
}

func (d *NonNullType) IsOutputType() bool {
	return d.Type.IsOutputType()
}

func (d *NonNullType) IsSubTypeOf(other Type) bool {
	return d.IsSameType(other) || d.Type.IsSubTypeOf(other)
}

func (d *NonNullType) IsSameType(other Type) bool {
	if nn, ok := other.(*NonNullType); ok {
		return d.Type.IsSameType(nn.Type)
	}
	return false
}

func isNonNull(t Type) bool {
	_, ok := t.(*NonNullType)
	return ok
}
