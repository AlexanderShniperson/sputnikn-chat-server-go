package server

import (
	pb "chatserver/contract/v1"
	"time"
)

type MessageToRoom struct {
	Message any
	OutChan chan any
}

type GetRoomDetail struct{}

type RoomDetailReply struct {
	Reply *pb.RoomDetail
}

type SetRoomReadMarker struct {
	UserId     string
	ReadMarker time.Time
}
