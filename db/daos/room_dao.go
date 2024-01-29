package daos

import (
	utils "chatserver/utils"
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/lo"

	entities "chatserver/db/entities"

	pb "chatserver/contract/v1"

	pgxuuid "github.com/jackc/pgx-gofrs-uuid"
)

type RoomDao struct {
	dbPool *pgxpool.Pool
}

var messageCondition = []string{
	pb.RoomEventType_roomEventTypeMessage.String(),
	pb.RoomEventType_roomEventTypeAll.String(),
}

func NewRoomDao(dbPool *pgxpool.Pool) *RoomDao {
	return &RoomDao{
		dbPool: dbPool,
	}
}

func (e *RoomDao) GetRooms() ([]*entities.RoomEntity, error) {
	query := "SELECT id, title, avatar FROM room"
	rows, err := e.dbPool.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]*entities.RoomEntity, 0)

	for rows.Next() {
		var roomUuid pgxuuid.UUID
		var roomTitle string
		var roomAvatar *string
		err = rows.Scan(&roomUuid, &roomTitle, &roomAvatar)
		if err != nil {
			return nil, err
		}

		roomUuidStr, err := utils.UuidToString(roomUuid)
		if err != nil {
			return nil, err
		}
		result = append(result, &entities.RoomEntity{
			RoomId: *roomUuidStr,
			Title:  roomTitle,
			Avatar: roomAvatar,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *RoomDao) GetRoomsByUserId(userId string) (map[string]*entities.RoomEntity, error) {
	query := `SELECT id, title, avatar 
	FROM room r
	INNER JOIN room_member rm ON rm.room_id = r.id
	INNER JOIN "user" u ON rm.user_id = u.id
	WHERE u.id = $1`
	rows, err := e.dbPool.Query(context.Background(), query, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]*entities.RoomEntity, 0)

	for rows.Next() {
		var roomUuid pgxuuid.UUID
		var roomTitle string
		var roomAvatar *string
		err = rows.Scan(&roomUuid, &roomTitle, &roomAvatar)
		if err != nil {
			return nil, err
		}

		roomUuidStr, err := utils.UuidToString(roomUuid)
		if err != nil {
			return nil, err
		}
		result[*roomUuidStr] = &entities.RoomEntity{
			RoomId: *roomUuidStr,
			Title:  roomTitle,
			Avatar: roomAvatar,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *RoomDao) GetRoomMembers(roomId string) ([]*entities.RoomMemberEntity, error) {
	query := `SELECT u.id, u.full_name, u.avatar, rm.member_status, rm.last_read_marker
	FROM room_member as rm
	INNER JOIN room as r ON r.id = rm.room_id
	INNER JOIN "user" as u ON u.id = rm.user_id
	WHERE r.id = $1`
	rows, err := e.dbPool.Query(context.Background(), query, roomId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]*entities.RoomMemberEntity, 0)

	for rows.Next() {
		var userUuid pgxuuid.UUID
		var userFullName string
		var userAvatar *string
		var memberStatus string
		var lastReadMarker *time.Time
		err = rows.Scan(&userUuid, &userFullName, &userAvatar, &memberStatus, &lastReadMarker)
		if err != nil {
			return nil, err
		}
		userUuidStr, err := utils.UuidToString(userUuid)
		if err != nil {
			return nil, err
		}
		result = append(result, &entities.RoomMemberEntity{
			UserId:         *userUuidStr,
			FullName:       userFullName,
			MemberStatus:   entities.ParseMemberStatus(memberStatus),
			Avatar:         userAvatar,
			IsOnline:       false,
			LastReadMarker: lastReadMarker,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *RoomDao) SetMemberReadMarker(roomId string, userId string, readMarker time.Time) error {
	query := `UPDATE room_member 
	SET last_read_marker = $1
	WHERE room_id = $2
	AND user_id = $3`
	_, err := e.dbPool.Exec(context.Background(), query, readMarker, roomId, userId)
	if err != nil {
		return err
	}
	return nil
}

func (e *RoomDao) GetSyncEvents(
	roomId string,
	eventType pb.RoomEventType,
	eventLimit int,
	sinceTime time.Time,
	orderType pb.SinceTimeOrderType) (*entities.RoomEventsEntity, error) {
	result := &entities.RoomEventsEntity{}
	//
	hasMessageCondition := lo.Contains[string](messageCondition, eventType.String())
	var whereCondition string
	if hasMessageCondition {
		switch orderType {
		case pb.SinceTimeOrderType_sinceTimeOrderTypeNewest:
			whereCondition = "date_create > $1"
		case pb.SinceTimeOrderType_sinceTimeOrderTypeOldest:
			whereCondition = "date_create < $1"
		}
	} else {
		whereCondition = "1=0"
	}
	queryMessageEventIds := fmt.Sprintf(`SELECT id, date_create
	FROM room_event_message
	WHERE %s
	ORDER BY date_create
	LIMIT $2
	`, whereCondition)
	rowsMessageEventIds, err := e.dbPool.Query(context.Background(), queryMessageEventIds, sinceTime, eventLimit)
	if err != nil {
		return nil, err
	}
	defer rowsMessageEventIds.Close()

	return result, nil
}
