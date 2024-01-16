package server

import (
	pb "chatserver/contract/v1"
	db "chatserver/db"
	dbentities "chatserver/db/entities"
	"fmt"
	"log"
)

type ChatRoom struct {
	database *db.SputnikDB
	Id       string
	title    string
	avatar   *string
	members  []*dbentities.RoomMemberEntity
	InChan   chan *MessageToRoom
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
				result := e.getRoomDetail()
				inMsg.OutChan <- &RoomDetailReply{
					Reply: result,
				}
			default:
				inMsg.OutChan <- fmt.Sprintf("unhandled message %T", v)
			}
		}
	}
}

func (e *ChatRoom) getRoomDetail() *pb.RoomDetail {
	members := make([]*pb.RoomMemberDetail, len(e.members))
	for idx, item := range e.members {
		members[idx] = &pb.RoomMemberDetail{
			UserId:       item.UserId,
			FullName:     item.FullName,
			IsOnline:     false,
			MemberStatus: pb.RoomMemberStatusType(item.MemberStatus),
		}
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

func (e *ChatRoom) initRoomMembers() {
	roomMembers, err := e.database.RoomDao.GetRoomMembers(e.Id)
	if err != nil {
		log.Fatal(err)
	}

	e.members = make([]*dbentities.RoomMemberEntity, 0)
	e.members = append(e.members, roomMembers...)
}
