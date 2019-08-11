package model

import (
	"crypto/sha512"
	"encoding/base64"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Id             Id
	RevisionNumber int

	Handle       string
	PasswordHash []byte
}

func PasswordHash(password string) []byte {
	h := sha512.Sum512([]byte(password))
	encoded := base64.RawURLEncoding.EncodeToString(h[:])
	hashed, err := bcrypt.GenerateFromPassword([]byte(encoded), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	return hashed
}

func VerifyPasswordHash(hash []byte, password string) bool {
	h := sha512.Sum512([]byte(password))
	encoded := base64.RawURLEncoding.EncodeToString(h[:])
	return bcrypt.CompareHashAndPassword(hash, []byte(encoded)) == nil
}
