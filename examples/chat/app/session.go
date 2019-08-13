package app

import (
	"github.com/sirupsen/logrus"

	"github.com/ccbrown/api-fu/examples/chat/model"
)

type Session struct {
	App    *App
	User   *model.User
	Logger logrus.FieldLogger
}

func (a *App) NewSession() *Session {
	return &Session{
		App:    a,
		Logger: logrus.StandardLogger(),
	}
}

func (s *Session) WithHandleAndPassword(handle, password string) (*Session, SanitizedError) {
	user, err := s.App.Store.GetUserByHandle(handle)
	if user == nil {
		return nil, s.InternalError(err)
	}
	if model.VerifyPasswordHash(user.PasswordHash, password) {
		ret := *s
		ret.User = user
		return &ret, nil
	}
	return nil, nil
}
