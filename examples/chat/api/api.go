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

const (
	UserTypeId    = 1
	ChannelTypeId = 2
	MessageTypeId = 3
)

func SerializeNodeId(typeId int, id model.Id) string {
	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutVarint(buf, int64(typeId))
	return base64.RawURLEncoding.EncodeToString(append(buf[:n], id...))
}

func DeserializeNodeId(id string) (int, model.Id) {
	if buf, err := base64.RawURLEncoding.DecodeString(id); err == nil {
		if typeId, n := binary.Varint(buf); n > 0 {
			return int(typeId), model.Id(buf[n:])
		}
	}
	return 0, nil
}

var fuCfg = apifu.Config{
	ResolveNodesByGlobalIds: func(ctx context.Context, ids []string) ([]interface{}, error) {
		var userIds []model.Id
		var channelIds []model.Id
		var messageIds []model.Id
		for _, id := range ids {
			typeId, id := DeserializeNodeId(id)
			switch typeId {
			case UserTypeId:
				userIds = append(userIds, id)
			case ChannelTypeId:
				channelIds = append(channelIds, id)
			case MessageTypeId:
				messageIds = append(messageIds, id)
			}
		}
		sess := ctxSession(ctx)
		channels, err := sess.GetChannelsByIds(channelIds...)
		if err != nil {
			return nil, err
		}
		messages, err := sess.GetMessagesByIds(messageIds...)
		if err != nil {
			return nil, err
		}
		users, err := sess.GetUsersByIds(userIds...)
		if err != nil {
			return nil, err
		}
		ret := make([]interface{}, 0, len(ids))
		for _, channel := range channels {
			ret = append(ret, channel)
		}
		for _, message := range messages {
			ret = append(ret, message)
		}
		for _, user := range users {
			ret = append(ret, user)
		}
		return ret, nil
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

func (api *API) ServeGraphQLWS(w http.ResponseWriter, r *http.Request) {
	api.init()
	api.withSession(api.fu.ServeGraphQLWS)(w, r)
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
