package model

import "time"

type Channel struct {
	Id             Id
	RevisionNumber int

	CreatorUserId Id
	CreationTime  time.Time
	Name          string
}
