package api

import (
	"context"
	"reflect"

	"github.com/ccbrown/api-fu"
	"github.com/ccbrown/api-fu/examples/chat/app"
	"github.com/ccbrown/api-fu/examples/chat/model"
)

func init() {
	apiFu.AddNode(&apifu.Node{
		TypeId: 1,
		Model:  reflect.TypeOf(model.User{}),
		GetByIds: func(ctx context.Context, ids interface{}) (interface{}, error) {
			return ctx.Value(sessionContextKey).(*app.Session).GetUsersByIds(ids.([]model.Id)...)
		},
		Fields: map[string]*apifu.NodeField{
			"handle": apifu.NonNullString("Handle"),
		},
	})
}
