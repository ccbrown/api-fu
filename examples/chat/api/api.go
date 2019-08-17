package api

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"net/http"
	"sync"

	apifu "github.com/ccbrown/api-fu"
	"github.com/ccbrown/api-fu/examples/chat/app"
	"github.com/ccbrown/api-fu/examples/chat/model"
)

func DeserializeId(id string) model.Id {
	if buf, err := base64.RawURLEncoding.DecodeString(id); err == nil {
		if _, n := binary.Varint(buf); n > 0 {
			return model.Id(buf[n:])
		}
	}
	return nil
}

var fuCfg = apifu.Config{
	SerializeNodeId: func(typeId int, id interface{}) string {
		buf := make([]byte, binary.MaxVarintLen64)
		n := binary.PutVarint(buf, int64(typeId))
		return base64.RawURLEncoding.EncodeToString(append(buf[:n], id.(model.Id)...))
	},
	DeserializeNodeId: func(id string) (int, interface{}) {
		if buf, err := base64.RawURLEncoding.DecodeString(id); err == nil {
			if typeId, n := binary.Varint(buf); n > 0 {
				return int(typeId), model.Id(buf[n:])
			}
		}
		return 0, nil
	},
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
	api.withSession(api.fu.ServeGraphQL)(w, r)
}

type sessionContextKeyType int

var sessionContextKey sessionContextKeyType

func ctxSession(ctx context.Context) *app.Session {
	return ctx.Value(sessionContextKey).(*app.Session)
}

func statusCodeForError(err error) int {
	switch err.(type) {
	case *app.UserError:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func (api *API) withSession(f func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		session := api.App.NewSession()

		if handle, password, ok := r.BasicAuth(); ok {
			if newSession, err := session.WithHandleAndPassword(handle, password); err != nil {
				http.Error(w, err.Error(), statusCodeForError(err))
				return
			} else if newSession == nil {
				http.Error(w, "Invalid credentials.", http.StatusUnauthorized)
				return
			} else {
				session = newSession
			}
		}

		r = r.WithContext(context.WithValue(r.Context(), sessionContextKey, session))
		f(w, r)
	}
}
