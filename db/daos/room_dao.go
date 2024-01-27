package daos

import (
	utils "chatserver/utils"
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	entities "chatserver/db/entities"

	pgxuuid "github.com/jackc/pgx-gofrs-uuid"
)

type RoomDao struct {
	dbPool *pgxpool.Pool
}

func NewRoomDao(dbPool *pgxpool.Pool) *RoomDao {
	return &RoomDao{
		dbPool: dbPool,
	}
}

func (e *RoomDao) GetRooms() ([]*entities.RoomEntity, error) {
	rows, err := e.dbPool.Query(context.Background(), "SELECT id, title, avatar FROM room")
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
