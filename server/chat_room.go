package server

import (
	pb "chatserver/contract/v1"
	"chatserver/db"
	"chatserver/db/entities"
	"errors"
	"fmt"
	"log"
	"time"
)

type ChatRoomOnlineUser struct {
	UserId string
	Stream pb.ChatStreamService_RoomEventStreamServer
}

type ChatRoom struct {
	database    *db.SputnikDB
	Id          string
	title       string
	avatar      *string
	members     map[string]*entities.RoomMemberEntity
	InChan      chan *MessageToRoom
	onlineUsers map[string]*ChatRoomOnlineUser
}

func CreateRoom(database *db.SputnikDB, roomId string, roomTitle string, roomAvatar *string) *ChatRoom {
	return &ChatRoom{
		database: database,
		Id:       roomId,
		title:    roomTitle,
		avatar:   roomAvatar,
		InChan:   make(chan *MessageToRoom),
	}
}

func (e *ChatRoom) Run() {
	log.Printf("[ChatRoom] started id=%s\n", e.Id)
	e.initRoomMembers()
	for {
		select {
		case inMsg := <-e.InChan:
			switch v := inMsg.Message.(type) {
			case *GetRoomDetail:
				result := e.buildRoomDetail()
				inMsg.OutChan <- &RoomDetailReply{
					Reply: result,
				}
			case *SetRoomReadMarker:
				e.setMemberReadMarker(v.UserId, v.ReadMarker)
				result := e.buildRoomDetail()
				inMsg.OutChan <- &RoomDetailReply{
					Reply: result,
				}
				e.sendBroadcastMessage(&pb.RoomEventResponse{
					Payload: &pb.RoomEventResponse_RoomStateChanged{
						RoomStateChanged: &pb.RoomStateChangedResponse{
							Detail: result,
						},
					},
				})
				// send broadcast message to all users
			default:
				inMsg.OutChan <- fmt.Sprintf("unhandled message %T", v)
			}
		}
	}
}

func (e *ChatRoom) initRoomMembers() {
	roomMembers, err := e.database.RoomDao.GetRoomMembers(e.Id)
	if err != nil {
		log.Fatal(err)
	}

	e.members = make(map[string]*entities.RoomMemberEntity)
	for _, item := range roomMembers {
		e.members[item.UserId] = item
	}
}

func (e *ChatRoom) buildRoomDetail() *pb.RoomDetail {
	members := make([]*pb.RoomMemberDetail, len(e.members))
	idx := 0
	for _, item := range e.members {
		var lastRead *int64
		if item.LastReadMarker != nil {
			dateMilli := item.LastReadMarker.UnixMilli()
			lastRead = &dateMilli
		}
		members[idx] = &pb.RoomMemberDetail{
			UserId:         item.UserId,
			FullName:       item.FullName,
			IsOnline:       item.IsOnline,
			MemberStatus:   pb.RoomMemberStatusType(item.MemberStatus),
			Avatar:         item.Avatar,
			LastReadMarker: lastRead,
		}
		idx++
	}
	return &pb.RoomDetail{
		RoomId:                  e.Id,
		Title:                   e.title,
		Avatar:                  e.avatar,
		Members:                 members,
		EventMessageUnreadCount: -1,
		EventSystemUnreadCount:  -1,
	}
}

func (e *ChatRoom) setMemberReadMarker(userId string, readMarker time.Time) error {
	err := e.database.RoomDao.SetMemberReadMarker(e.Id, userId, readMarker)
	if err != nil {
		return err
	}

	if user, ok := e.members[userId]; ok {
		user.LastReadMarker = &readMarker
		return nil
	}

	return errors.New("user not found")
}

func (e *ChatRoom) sendBroadcastMessage(message *pb.RoomEventResponse) {
	for _, item := range e.onlineUsers {
		item.Stream.Send(message)
	}
}
