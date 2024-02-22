package server

import (
	pb "chatserver/contract/v1"
	"errors"

	"github.com/samber/lo"
)

type ChatStreamService struct {
	pb.UnimplementedChatStreamServiceServer
	roomManager *RoomManager
	onlineUsers []*pb.ChatStreamService_RoomEventStreamServer
}

func NewChatStreamService(roomManager *RoomManager) *ChatStreamService {
	return &ChatStreamService{
		roomManager: roomManager,
		onlineUsers: make([]*pb.ChatStreamService_RoomEventStreamServer, 0),
	}
}

func (e *ChatStreamService) GetOnlineUsers() []*pb.ChatStreamService_RoomEventStreamServer {
	return lo.Compact(e.onlineUsers)
}

func (e *ChatStreamService) RoomEventStream(req *pb.EmptyRequest, client pb.ChatStreamService_RoomEventStreamServer) error {
	return errors.New("error")
}
