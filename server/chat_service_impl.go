package server

import (
	pb "chatserver/contract/v1"
	"context"
	"errors"
	"log"
	"sync"

	db "chatserver/db"
)

type chatServiceImpl struct {
	pb.UnimplementedChatServiceServer
	tokenManager *JWTManager
	database     *db.SputnikDB
	roomManager  *RoomManager
}

func CreateNewChatService(database *db.SputnikDB, tokenManager *JWTManager, roomManager *RoomManager) *chatServiceImpl {
	return &chatServiceImpl{
		database:     database,
		tokenManager: tokenManager,
		roomManager:  roomManager,
	}
}

func (e *chatServiceImpl) AuthUser(ctx context.Context, req *pb.AuthUserRequest) (*pb.AuthUserResponse, error) {
	foundUser, err := e.database.UserDao.FindUserByLoginPassword(req.Login, req.Password)

	if err != nil || foundUser == nil {
		result := &pb.AuthUserResponse{
			Error:       pb.AuthErrorType_AuthErrorTypeUserWrongCreds,
			AccessToken: nil,
		}
		return result, nil
	}

	tokenString, err := e.tokenManager.CreateToken(foundUser.Id)
	if err != nil {
		result := &pb.AuthUserResponse{
			Error:       pb.AuthErrorType_AuthErrorTypeUserWrongCreds,
			AccessToken: nil,
		}
		return result, nil
	}

	result := &pb.AuthUserResponse{
		Error:       pb.AuthErrorType_AuthErrorTypeNone,
		AccessToken: tokenString,
		Detail: &pb.UserDetail{
			UserId:   foundUser.Id,
			FullName: foundUser.FullName,
			Avatar:   foundUser.Avatar,
		},
	}
	log.Printf(">>> User=%s AuthToken=%s", req.Login, *tokenString)
	return result, nil
}

func (e *chatServiceImpl) ListRooms(ctx context.Context, req *pb.ListRoomsRequest) (*pb.ListRoomsResponse, error) {
	rooms := e.roomManager.GetRooms()
	roomsCount := len(rooms)
	var wg sync.WaitGroup
	roomDetails := make([]*pb.RoomDetail, roomsCount)
	wg.Add(roomsCount)
	idx := 0
	for _, room := range rooms {
		inChan := room.InChan
		outChan := make(chan any)
		go func(idx int) {
			defer wg.Done()
			inChan <- &MessageToRoom{
				Message: &GetRoomDetail{},
				OutChan: outChan,
			}
			msg := <-outChan
			if result, ok := msg.(*RoomDetailReply); ok {
				roomDetails[idx] = result.Reply
			}
		}(idx)
		idx += 1
	}
	wg.Wait()
	resp := &pb.ListRoomsResponse{
		Detail: roomDetails,
	}
	return resp, nil
}

func (s *chatServiceImpl) SyncRooms(ctx context.Context, req *pb.SyncRoomsRequest) (*pb.SyncRoomsResponse, error) {
	return nil, errors.New("Error")
}

func (s *chatServiceImpl) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	return nil, errors.New("Error")
}

func (s *chatServiceImpl) SetRoomReadMarker(ctx context.Context, req *pb.RoomReadMarkerRequest) (*pb.RoomStateChangedResponse, error) {
	return nil, errors.New("Error")
}

func (s *chatServiceImpl) CreateRoom(ctx context.Context, req *pb.CreateRoomRequest) (*pb.CreateRoomResponse, error) {
	return nil, errors.New("Error")
}

func (s *chatServiceImpl) InviteRoomMember(ctx context.Context, req *pb.EmptyRequest) (*pb.RoomStateChangedResponse, error) {
	return nil, errors.New("Error")
}

func (s *chatServiceImpl) RemoveRoomMember(ctx context.Context, req *pb.EmptyRequest) (*pb.RoomStateChangedResponse, error) {
	return nil, errors.New("Error")
}

func (s *chatServiceImpl) AddRoomMessage(ctx context.Context, req *pb.RoomEventMessageRequest) (*pb.RoomEventMessageResponse, error) {
	return nil, errors.New("Error")
}
