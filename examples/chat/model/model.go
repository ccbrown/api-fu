package model

import (
	"bytes"
	"crypto/rand"
)

// Id represents a node id. Ids are cryptographically secure random byte strings long enough for us
// to generate them without having worry about collisions with pre-existing ids.
type Id []byte

func (id Id) Before(other Id) bool {
	return bytes.Compare(id, other) == -1
}

func (id Id) MarshalBinary() ([]byte, error) {
	return id, nil
}

// GenerateId generates a new Id.
func GenerateId() Id {
	id := make(Id, 20)
	if _, err := rand.Read(id); err != nil {
		panic(err)
	}
	return id
}
