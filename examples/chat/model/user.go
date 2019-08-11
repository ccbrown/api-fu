package model

type User struct {
	Id             Id
	RevisionNumber int

	Handle       string
	PasswordHash []byte
}
