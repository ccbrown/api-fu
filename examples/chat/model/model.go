package model

import "crypto/rand"

type Id []byte

func GenerateId() Id {
	id := make(Id, 20)
	if _, err := rand.Read(id); err != nil {
		panic(err)
	}
	return id
}
