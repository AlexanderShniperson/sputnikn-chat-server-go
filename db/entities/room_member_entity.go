package entities

import "time"

type RoomMemberEntity struct {
	UserId         string
	FullName       string
	MemberStatus   MemberStatus
	Avatar         *string
	IsOnline       bool
	LastReadMarker *time.Time
}
