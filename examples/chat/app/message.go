package app

import (
	"time"

	"github.com/ccbrown/api-fu/examples/chat/model"
)

func (s *Session) CreateMessage(input *model.Message) (*model.Message, SanitizedError) {
	if s.User == nil {
		return nil, s.AuthorizationError()
	}

	message := *input
	message.Id = model.GenerateId()
	message.UserId = s.User.Id
	message.Time = time.Now()
	message.RevisionNumber = 1

	if channel, err := s.GetChannelById(message.ChannelId); err != nil {
		return nil, err
	} else if channel == nil {
		return nil, s.UserError("Invalid channel id.")
	} else if message.Body == "" {
		return nil, s.UserError("A body is required.")
	} else if err := s.App.Store.AddMessage(&message); err != nil {
		return nil, s.InternalError(err)
	}

	return &message, nil
}

func (s *Session) GetMessagesByIds(ids ...model.Id) ([]*model.Message, SanitizedError) {
	messages, err := s.App.Store.GetMessagesByIds(ids...)
	return messages, s.InternalError(err)
}

// GetMessagesByChannelIdAndTimeRange gets messages for a particular channel within an inclusive
// time range. If limit is non-zero, the returned messages will be limited to that number. If limit
// is negative, the returned messages will be the last messages in the range.
func (s *Session) GetMessagesByChannelIdAndTimeRange(channelId model.Id, begin, end time.Time, limit int) ([]*model.Message, error) {
	messages, err := s.App.Store.GetMessagesByChannelIdAndTimeRange(channelId, begin, end, limit)
	return messages, s.InternalError(err)
}
