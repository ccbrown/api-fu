package api

import (
	"net/http"
	"sync"

	"github.com/ccbrown/api-fu"
	"github.com/ccbrown/api-fu/examples/chat/app"
)

var apiFu apifu.Config

type sessionContextKeyType int

var sessionContextKey sessionContextKeyType

type API struct {
	App *app.App

	fu       *apifu.API
	initOnce sync.Once
}

func (api *API) init() {
	api.initOnce.Do(func() {
		fu, err := apifu.NewAPI(&apiFu)
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
