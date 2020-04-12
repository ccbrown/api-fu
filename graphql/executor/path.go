package executor

type path struct {
	Prev            *path
	StringComponent string
	IntComponent    int
}

func (p *path) WithIntComponent(n int) *path {
	return &path{
		Prev:         p,
		IntComponent: n,
	}
}

func (p *path) WithStringComponent(s string) *path {
	return &path{
		Prev:            p,
		StringComponent: s,
	}
}

func (p *path) Slice() []interface{} {
	if p == nil {
		return nil
	}
	if p.StringComponent != "" {
		return append(p.Prev.Slice(), p.StringComponent)
	}
	return append(p.Prev.Slice(), p.IntComponent)
}
