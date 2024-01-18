package server

import (
	"errors"
	"log"

	db "chatserver/db"
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

func (e *RoomManager) FindRoom(roomId string) (*ChatRoom, error) {
	for k, v := range e.rooms {
		if k == roomId {
			return v, nil
		}
	}
	return nil, errors.New("not found")
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
