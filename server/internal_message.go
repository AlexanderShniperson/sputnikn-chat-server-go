package server

import (
	pb "chatserver/contract/v1"
	"time"
)

type MessageToRoom struct {
	Message any
	OutChan chan any
}

type GetRoomDetailInternal struct{}

type RoomDetailReplyInternal struct {
	Reply *pb.RoomDetail
}

type SetRoomReadMarkerInternal struct {
	UserId     string
	ReadMarker time.Time
}

type SyncRoomEventsInternal struct {
	UserId string
	Filter *pb.SyncRoomFilter
}

type SyncRoomEventsReplyInternal struct {
	RoomId        string
	MessageEvents []*pb.RoomEventMessageDetail
	SystemEvents  []*pb.RoomEventSystemDetail
}
