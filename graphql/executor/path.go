package executor

type path struct {
	Prev      *path
	Component interface{}
}

func (p *path) WithComponent(component interface{}) *path {
	return &path{
		Prev:      p,
		Component: component,
	}
}

func (p *path) Slice() []interface{} {
	if p == nil {
		return nil
	}
	return append(p.Prev.Slice(), p.Component)
}
