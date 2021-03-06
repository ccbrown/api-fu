package store

import (
	"fmt"

	"github.com/ccbrown/api-fu/examples/chat/model"
)

var ErrUserHandleExists = fmt.Errorf("user handle exists")

// Adds a user to the database. Returns ErrUserHandleExists if the handle is taken.
func (s *Store) AddUser(user *model.User) error {
	serialized, err := serialize(user)
	if err != nil {
		return err
	}

	tx := s.Backend.AtomicWrite()
	tx.Set("user:"+string(user.Id), serialized)
	tx.SetNX("user_by_handle:"+user.Handle, user.Id)
	if didCommit, err := tx.Exec(); err != nil {
		return err
	} else if !didCommit {
		return ErrUserHandleExists
	}
	return nil
}

func (s *Store) GetUsersByIds(ids ...model.Id) ([]*model.User, error) {
	var ret []*model.User
	return ret, s.getByIds("user", &ret, ids...)
}

func (s *Store) GetUserByHandle(handle string) (*model.User, error) {
	id, err := s.Backend.Get("user_by_handle:" + handle)
	if id == nil {
		return nil, err
	}
	users, err := s.GetUsersByIds(model.Id(*id))
	if len(users) < 1 {
		return nil, err
	}
	return users[0], nil
}
