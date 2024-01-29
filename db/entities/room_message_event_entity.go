package entities

import "time"

type RoomMessageEventEntity struct {
	Id            string
	RoomId        string
	UserId        string
	ClientEventId string
	Version       int
	Content       string
	DateCreate    time.Time
	DateUpdate    *time.Time
}
