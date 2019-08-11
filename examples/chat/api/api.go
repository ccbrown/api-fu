package api

import (
	"context"
	"net/http"
	"sync"

	"github.com/ccbrown/api-fu"
	"github.com/ccbrown/api-fu/examples/chat/app"
)

var fuCfg apifu.Config

type sessionContextKeyType int

var sessionContextKey sessionContextKeyType

func ctxSession(ctx context.Context) *app.Session {
	return ctx.Value(sessionContextKey).(*app.Session)
}

type API struct {
	App *app.App

	fu       *apifu.API
	initOnce sync.Once
}

func (api *API) init() {
	api.initOnce.Do(func() {
		fu, err := apifu.NewAPI(&fuCfg)
		if err != nil {
			panic(err)
		}
		api.fu = fu
	})
}

func (api *API) ServeGraphQL(w http.ResponseWriter, r *http.Request) {
	api.init()
	api.fu.ServeGraphQL(w, r)
}
