package jsonapi

import "context"

type ConstantString[T any] string

func (c ConstantString[T]) ResolveAttribute(ctx context.Context, resource T) (any, *Error) {
	return c, nil
}
