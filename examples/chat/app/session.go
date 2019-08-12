package app

import (
	"github.com/sirupsen/logrus"

	"github.com/ccbrown/api-fu/examples/chat/model"
)

type Session struct {
	App    *App
	UserId model.Id
	Logger logrus.FieldLogger
}

func (a *App) NewSession() *Session {
	return &Session{
		App:    a,
		Logger: logrus.StandardLogger(),
	}
}
