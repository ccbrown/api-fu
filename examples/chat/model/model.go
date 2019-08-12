package model

import "crypto/rand"

type Id []byte

func (id Id) MarshalBinary() ([]byte, error) {
	return id, nil
}

func GenerateId() Id {
	id := make(Id, 20)
	if _, err := rand.Read(id); err != nil {
		panic(err)
	}
	return id
}
