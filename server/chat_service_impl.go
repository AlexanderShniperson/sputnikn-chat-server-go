package server

import (
	pb "chatserver/contract/v1"
	"chatserver/utils"
	"context"
	"errors"
	"log"
	"sync"
	"time"

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
	log.Printf(">>> User=%s AuthToken=%s\n", req.Login, *tokenString)
	return result, nil
}

func (e *chatServiceImpl) ListRooms(ctx context.Context, req *pb.ListRoomsRequest) (*pb.ListRoomsResponse, error) {
	rooms := e.roomManager.GetRooms(req.RoomIds)
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

func (e *chatServiceImpl) SyncRooms(ctx context.Context, req *pb.SyncRoomsRequest) (*pb.SyncRoomsResponse, error) {
	return nil, errors.New("Error")
}

func (e *chatServiceImpl) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	users, err := e.database.UserDao.GetAllUsers()
	if err != nil {
		log.Println("ListUsers error:", err)
		return nil, errors.New("internal error")
	}

	userDetails := make([]*pb.UserDetail, len(users))
	for idx, user := range users {
		userDetails[idx] = &pb.UserDetail{
			UserId:   user.Id,
			FullName: user.FullName,
			Avatar:   user.Avatar,
		}
	}

	result := &pb.ListUsersResponse{
		Users: userDetails,
	}

	return result, nil
}

func (e *chatServiceImpl) SetRoomReadMarker(ctx context.Context, req *pb.RoomReadMarkerRequest) (*pb.RoomStateChangedResponse, error) {
	foundRoom, err := e.roomManager.FindRoom(req.RoomId)
	if err != nil {
		return nil, errors.New("room not found")
	}
	accessToken, err := utils.GetAccessTokenFromContext(ctx)
	if err != nil {
		return nil, errors.New("internal error")
	}
	userClaims, err := e.tokenManager.VerifyToken(*accessToken)
	if err != nil {
		return nil, errors.New("internal error")
	}
	outChan := make(chan any)
	foundRoom.InChan <- &MessageToRoom{
		Message: &SetRoomReadMarker{
			UserId:     userClaims.UserId,
			ReadMarker: time.UnixMilli(req.ReadMarkerTimestamp),
		},
	}
	msg := <-outChan
	if reply, ok := msg.(*RoomDetailReply); ok {
		result := &pb.RoomStateChangedResponse{
			Detail: reply.Reply,
		}
		return result, nil
	}
	return nil, errors.New("internal error")
}

func (e *chatServiceImpl) CreateRoom(ctx context.Context, req *pb.CreateRoomRequest) (*pb.CreateRoomResponse, error) {
	return nil, errors.New("Error")
}

func (e *chatServiceImpl) InviteRoomMember(ctx context.Context, req *pb.EmptyRequest) (*pb.RoomStateChangedResponse, error) {
	return nil, errors.New("Error")
}

func (e *chatServiceImpl) RemoveRoomMember(ctx context.Context, req *pb.EmptyRequest) (*pb.RoomStateChangedResponse, error) {
	return nil, errors.New("Error")
}

func (e *chatServiceImpl) AddRoomMessage(ctx context.Context, req *pb.RoomEventMessageRequest) (*pb.RoomEventMessageResponse, error) {
	return nil, errors.New("Error")
}
