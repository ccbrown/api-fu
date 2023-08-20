package jsonapi

import (
	"context"

	"github.com/ccbrown/api-fu/jsonapi/types"
)

type ConstantString[T any] string

func (c ConstantString[T]) ResolveAttribute(ctx context.Context, resource T) (any, *types.Error) {
	return c, nil
}
