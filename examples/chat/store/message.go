package store

import (
	"time"

	"github.com/ccbrown/api-fu/examples/chat/model"
)

func (s *Store) AddMessage(message *model.Message) error {
	serialized, err := serialize(message)
	if err != nil {
		return err
	}

	tx := s.Backend.AtomicWrite()
	tx.Set("message:"+string(message.Id), serialized)
	tx.ZAdd("messages_by_channel:"+string(message.ChannelId), message.Id, float64(message.Time.UnixNano()))
	_, err = tx.Exec()
	return err
}

func (s *Store) GetMessagesByIds(ids ...model.Id) ([]*model.Message, error) {
	var ret []*model.Message
	return ret, s.getByIds("message", &ret, ids...)
}

// GetMessagesByChannelIdAndTimeRange gets messages for a particular channel within an inclusive
// time range. If limit is non-zero, the returned messages will be limited to that number. If limit
// is negative, the returned messages will be the last messages in the range.
func (s *Store) GetMessagesByChannelIdAndTimeRange(channelId model.Id, begin, end time.Time, limit int) ([]*model.Message, error) {
	zrange := s.Backend.ZRangeByScore
	if limit < 0 {
		zrange = s.Backend.ZRevRangeByScore
		limit = -limit
	}
	ids, err := zrange("messages_by_channel:"+string(channelId), float64(begin.UnixNano()), float64(end.UnixNano()), limit)
	if err != nil {
		return nil, err
	}
	return s.GetMessagesByIds(stringsToIds(ids)...)
}
