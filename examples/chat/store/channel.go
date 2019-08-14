package store

import (
	"github.com/ccbrown/api-fu/examples/chat/model"
)

func (s *Store) AddChannel(channel *model.Channel) error {
	serialized, err := serialize(channel)
	if err != nil {
		return err
	}

	tx := s.Backend.AtomicWrite()
	tx.Set("channel:"+string(channel.Id), serialized)
	tx.SAdd("channels", channel.Id)
	_, err = tx.Exec()
	return err
}

func (s *Store) GetChannelsByIds(ids ...model.Id) ([]*model.Channel, error) {
	var ret []*model.Channel
	return ret, s.getByIds("channel", &ret, ids...)
}

func (s *Store) GetChannels() ([]*model.Channel, error) {
	ids, err := s.Backend.SMembers("channels")
	if ids == nil {
		return nil, err
	}
	return s.GetChannelsByIds(stringsToIds(ids)...)
}
