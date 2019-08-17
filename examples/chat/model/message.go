package model

import "time"

type Message struct {
	Id             Id
	RevisionNumber int

	UserId    Id
	ChannelId Id
	Time      time.Time
	Body      string
}
