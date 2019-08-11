package app

import (
	"github.com/ccbrown/api-fu/examples/chat/model"
	"github.com/ccbrown/api-fu/examples/chat/store"
)

func (s *Session) CreateUser(input *model.User) (*model.User, SanitizedError) {
	user := *input
	user.Id = model.GenerateId()
	user.RevisionNumber = 1

	if user.Handle == "" {
		return nil, s.UserError("A handle is required.")
	} else if len(user.PasswordHash) == 0 {
		return nil, s.UserError("A password is required.")
	} else if err := s.App.Store.AddUser(&user); err == store.ErrUserHandleExists {
		return nil, s.UserError("That handle is already in use.")
	} else if err != nil {
		return nil, s.InternalError(err)
	}

	return &user, nil
}

func (s *Session) GetUsersByIds(ids ...model.Id) ([]*model.User, SanitizedError) {
	users, err := s.App.Store.GetUsersByIds(ids...)
	return users, s.InternalError(err)
}
