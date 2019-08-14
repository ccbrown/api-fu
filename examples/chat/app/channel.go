package app

import (
	"time"

	"github.com/ccbrown/api-fu/examples/chat/model"
)

func (s *Session) CreateChannel(input *model.Channel) (*model.Channel, SanitizedError) {
	if s.User == nil {
		return nil, s.AuthorizationError()
	}

	channel := *input
	channel.Id = model.GenerateId()
	channel.CreatorUserId = s.User.Id
	channel.CreationTime = time.Now()
	channel.RevisionNumber = 1

	if channel.Name == "" {
		return nil, s.UserError("A name is required.")
	} else if err := s.App.Store.AddChannel(&channel); err != nil {
		return nil, s.InternalError(err)
	}

	return &channel, nil
}

func (s *Session) GetChannelsByIds(ids ...model.Id) ([]*model.Channel, SanitizedError) {
	channels, err := s.App.Store.GetChannelsByIds(ids...)
	return channels, s.InternalError(err)
}

func (s *Session) GetChannels() ([]*model.Channel, SanitizedError) {
	channels, err := s.App.Store.GetChannels()
	return channels, s.InternalError(err)
}
