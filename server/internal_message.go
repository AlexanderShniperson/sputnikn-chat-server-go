package server

import (
	pb "chatserver/contract/v1"
)

type MessageToRoom struct {
	Message any
	OutChan chan any
}

type GetRoomDetail struct{}

type RoomDetailReply struct {
	Reply *pb.RoomDetail
}
