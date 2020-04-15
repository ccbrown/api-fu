package store

import (
	"reflect"

	"github.com/ccbrown/keyvaluestore"
	"github.com/vmihailenco/msgpack"

	"github.com/ccbrown/api-fu/examples/chat/model"
)

// Store implements the persistence layer of our application.
type Store struct {
	Backend keyvaluestore.Backend
}

func serialize(v interface{}) (string, error) {
	b, err := msgpack.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func deserialize(s string, dest interface{}) error {
	return msgpack.Unmarshal([]byte(s), dest)
}

func (s *Store) getByIds(key string, dest interface{}, ids ...model.Id) error {
	batch := s.Backend.Batch()
	gets := make([]keyvaluestore.GetResult, 0, len(ids))
	keys := map[string]struct{}{}
	for _, id := range ids {
		key := key + ":" + string(id)
		if _, ok := keys[key]; !ok {
			gets = append(gets, batch.Get(key))
			keys[key] = struct{}{}
		}
	}
	if err := batch.Exec(); err != nil {
		return err
	}

	objType := reflect.TypeOf(dest).Elem().Elem().Elem()
	slice := reflect.ValueOf(dest).Elem()
	for _, get := range gets {
		if v, _ := get.Result(); v != nil {
			obj := reflect.New(objType)
			if err := deserialize(*v, obj.Interface()); err != nil {
				return err
			}
			slice = reflect.Append(slice, obj)
		}
	}
	reflect.ValueOf(dest).Elem().Set(slice)
	return nil
}

func stringsToIds(s []string) []model.Id {
	ret := make([]model.Id, len(s))
	for i, id := range s {
		ret[i] = model.Id(id)
	}
	return ret
}
