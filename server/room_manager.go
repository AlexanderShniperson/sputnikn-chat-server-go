package server

import (
	"log"

	db "chatserver/db"

	"github.com/samber/mo"
)

type RoomManager struct {
	database *db.SputnikDB
	rooms    map[string]*ChatRoom
}

func CreateRoomManager(database *db.SputnikDB) *RoomManager {
	return &RoomManager{
		database: database,
		rooms:    make(map[string]*ChatRoom),
	}
}

func (e *RoomManager) GetRooms(roomIds []string) map[string]*ChatRoom {
	result := make(map[string]*ChatRoom)
	for k, v := range e.rooms {
		if len(roomIds) == 0 {
			result[k] = v
		} else {
			for _, roomId := range roomIds {
				if k == roomId {
					result[k] = v
				}
			}
		}
	}
	return result
}

func (e *RoomManager) FindRoom(roomId string) mo.Option[*ChatRoom] {
	var result *ChatRoom
	if room, ok := e.rooms[roomId]; ok {
		result = room
	}
	return mo.PointerToOption[*ChatRoom](&result)
}

func (e *RoomManager) Start() {
	rooms, err := e.database.RoomDao.GetRooms()
	if err != nil {
		log.Fatal(err)
	}

	for _, room := range rooms {
		createdRoom := CreateRoom(e.database, room.RoomId, room.Title, room.Avatar)
		e.rooms[room.RoomId] = createdRoom
		go createdRoom.Run()
	}
}
