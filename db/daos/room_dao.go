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

var (
	messageEventsCondition = []string{
		pb.RoomEventType_roomEventTypeMessage.String(),
		pb.RoomEventType_roomEventTypeAll.String(),
	}
	systemEventsCondition = []string{
		pb.RoomEventType_roomEventTypeSystem.String(),
		pb.RoomEventType_roomEventTypeAll.String(),
	}
)

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
	query := `SELECT r.id, r.title, r.avatar 
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
	FROM room_member rm
	INNER JOIN room r ON r.id = rm.room_id
	INNER JOIN "user" u ON u.id = rm.user_id
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

	messageEvents, err := e.getSyncMessageEvents(roomId, eventType, eventLimit, sinceTime, orderType)
	if err != nil {
		return nil, err
	}
	messageEventIds := lo.Map[*entities.RoomMessageEventEntity, string](
		messageEvents,
		func(event *entities.RoomMessageEventEntity, index int) string {
			return event.Id
		})

	attachmentEvents, err := e.getSyncMessageAttachmentEvents(messageEventIds, sinceTime, orderType)
	if err != nil {
		return nil, err
	}

	reactionEvents, err := e.getSyncMessageReactionEvents(messageEventIds, sinceTime, orderType)
	if err != nil {
		return nil, err
	}

	systemEvents, err := e.getSyncSystemEvents(roomId, eventType, eventLimit, sinceTime, orderType)
	if err != nil {
		return nil, err
	}

	result := &entities.RoomEventsEntity{
		RoomId:           roomId,
		MessageEvents:    messageEvents,
		AttachmentEvents: attachmentEvents,
		ReactionEvents:   reactionEvents,
		SystemEvents:     systemEvents,
	}
	return result, nil
}

func (e *RoomDao) getSyncMessageEvents(
	roomId string,
	eventType pb.RoomEventType,
	eventLimit int,
	sinceTime time.Time,
	orderType pb.SinceTimeOrderType) ([]*entities.RoomMessageEventEntity, error) {
	result := make([]*entities.RoomMessageEventEntity, 0)

	hasMessageEventsCondition := lo.Contains[string](messageEventsCondition, eventType.String())

	if hasMessageEventsCondition {
		var dateCondition string
		switch orderType {
		case pb.SinceTimeOrderType_sinceTimeOrderTypeNewest:
			dateCondition = "rem.date_create > $2"
		case pb.SinceTimeOrderType_sinceTimeOrderTypeOldest:
			dateCondition = "rem.date_create < $2"
		}
		query := fmt.Sprintf(
			`SELECT rem.id, rem.user_id, rem.client_event_id, rem.version, rem.content, rem.date_create, rem.date_edit
			FROM room_event_message rem
			INNER JOIN room r ON rem.room_id = r.id
			WHERE r.id = $1
			AND %s
			ORDER BY rem.date_create DESC
			LIMIT $3`, dateCondition)
		rows, err := e.dbPool.Query(context.Background(), query, roomId, sinceTime, eventLimit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var eventUuid pgxuuid.UUID
			var userUuid pgxuuid.UUID
			var clientEventId int
			var version int
			var content string
			var dateCreate time.Time
			var dateUpdate *time.Time
			err = rows.Scan(&eventUuid, &userUuid, &clientEventId, &version, &content, &dateCreate, &dateUpdate)
			if err != nil {
				return nil, err
			}
			eventUuidStr, err := utils.UuidToString(eventUuid)
			if err != nil {
				return nil, err
			}
			userUuidStr, err := utils.UuidToString(userUuid)
			if err != nil {
				return nil, err
			}
			result = append(result, &entities.RoomMessageEventEntity{
				Id:            *eventUuidStr,
				RoomId:        roomId,
				UserId:        *userUuidStr,
				ClientEventId: clientEventId,
				Version:       version,
				Content:       content,
				DateCreate:    dateCreate,
				DateUpdate:    dateUpdate,
			})
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (e *RoomDao) getSyncSystemEvents(
	roomId string,
	eventType pb.RoomEventType,
	eventLimit int,
	sinceTime time.Time,
	orderType pb.SinceTimeOrderType) ([]*entities.RoomSystemEventEntity, error) {
	result := make([]*entities.RoomSystemEventEntity, 0)

	hasSystemEventsCondition := lo.Contains[string](systemEventsCondition, eventType.String())

	if hasSystemEventsCondition {
		var dateCondition string
		switch orderType {
		case pb.SinceTimeOrderType_sinceTimeOrderTypeNewest:
			dateCondition = "res.date_create > $2"
		case pb.SinceTimeOrderType_sinceTimeOrderTypeOldest:
			dateCondition = "res.date_create < $2"
		}
		query := fmt.Sprintf(
			`SELECT res.id, res.version, res.content, res.date_create
		FROM room_event_system res
		INNER JOIN room r ON res.room_id = r.id
		WHERE r.id = $1
		AND %s
		ORDER BY res.date_create DESC
		LIMIT $3`, dateCondition)
		rows, err := e.dbPool.Query(context.Background(), query, roomId, sinceTime, eventLimit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var eventUuid pgxuuid.UUID
			var version int
			var content string
			var dateCreate time.Time
			err = rows.Scan(&eventUuid, &version, &content, &dateCreate)
			if err != nil {
				return nil, err
			}
			eventUuidStr, err := utils.UuidToString(eventUuid)
			if err != nil {
				return nil, err
			}
			result = append(result, &entities.RoomSystemEventEntity{
				Id:         *eventUuidStr,
				RoomId:     roomId,
				Version:    version,
				Content:    content,
				DateCreate: dateCreate,
			})
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (e *RoomDao) getSyncMessageAttachmentEvents(
	eventIds []string,
	sinceTime time.Time,
	orderType pb.SinceTimeOrderType) ([]*entities.RoomMessageEventAttachmentEntity, error) {
	result := make([]*entities.RoomMessageEventAttachmentEntity, 0)

	// TODO: potentially bug, because we can lost some events
	var dateCondition string
	switch orderType {
	case pb.SinceTimeOrderType_sinceTimeOrderTypeNewest:
		dateCondition = "rema.date_create > $2"
	case pb.SinceTimeOrderType_sinceTimeOrderTypeOldest:
		dateCondition = "rema.date_create < $2"
	}

	query := fmt.Sprintf(`SELECT rema.id, r.id, rem.id, ca.mime_type, rema.date_create
	FROM room_event_message_attachment rema
	INNER JOIN room_event_message rem ON rema.room_event_message_id = rem.id
	INNER JOIN room r ON rem.room_id = r.id
	INNER JOIN chat_attachment ca ON rema.chat_attachment_id = ca.id
	WHERE rem.id = ANY($1)
	AND %s
	ORDER BY rema.date_create DESC`, dateCondition)

	rows, err := e.dbPool.Query(context.Background(), query, eventIds, sinceTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var attachmentUuid pgxuuid.UUID
		var roomUuid pgxuuid.UUID
		var messageEventUuid pgxuuid.UUID
		var mimeType string
		var dateCreate time.Time
		err := rows.Scan(&attachmentUuid, &roomUuid, &messageEventUuid, &mimeType, &dateCreate)
		if err != nil {
			return nil, err
		}
		attachmentUuidStr, err := utils.UuidToString(attachmentUuid)
		if err != nil {
			return nil, err
		}
		roomUuidStr, err := utils.UuidToString(roomUuid)
		if err != nil {
			return nil, err
		}
		messageEventUuidStr, err := utils.UuidToString(messageEventUuid)
		if err != nil {
			return nil, err
		}
		result = append(result, &entities.RoomMessageEventAttachmentEntity{
			Id:             *attachmentUuidStr,
			RoomId:         *roomUuidStr,
			MessageEventId: *messageEventUuidStr,
			MimeType:       mimeType,
			DateCreate:     dateCreate,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *RoomDao) getSyncMessageReactionEvents(
	eventIds []string,
	sinceTime time.Time,
	orderType pb.SinceTimeOrderType) ([]*entities.RoomMessageEventReactionEntity, error) {
	result := make([]*entities.RoomMessageEventReactionEntity, 0)

	// TODO: potentially bug, because we can lost some events
	var dateCondition string
	switch orderType {
	case pb.SinceTimeOrderType_sinceTimeOrderTypeNewest:
		dateCondition = "remr.date_create > $2"
	case pb.SinceTimeOrderType_sinceTimeOrderTypeOldest:
		dateCondition = "remr.date_create < $2"
	}

	query := fmt.Sprintf(`SELECT remr.id, r.id, rem.id, remr.user_id, remr.content, remr.date_create
	FROM room_event_message_reaction remr
	INNER JOIN room_event_message rem ON remr.room_event_message_id = rem.id
	INNER JOIN room r ON rem.room_id = r.id
	WHERE rem.id = ANY($1)
	AND %s
	ORDER BY remr.date_create DESC`, dateCondition)

	rows, err := e.dbPool.Query(context.Background(), query, eventIds, sinceTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var reactionUuid pgxuuid.UUID
		var roomUuid pgxuuid.UUID
		var messageEventUuid pgxuuid.UUID
		var userUuid pgxuuid.UUID
		var content string
		var dateCreate time.Time
		err := rows.Scan(&reactionUuid, &roomUuid, &messageEventUuid, &userUuid, &content, &dateCreate)
		if err != nil {
			return nil, err
		}
		reactionUuidStr, err := utils.UuidToString(reactionUuid)
		if err != nil {
			return nil, err
		}
		roomUuidStr, err := utils.UuidToString(roomUuid)
		if err != nil {
			return nil, err
		}
		messageEventUuidStr, err := utils.UuidToString(messageEventUuid)
		if err != nil {
			return nil, err
		}
		userUuidStr, err := utils.UuidToString(userUuid)
		if err != nil {
			return nil, err
		}
		result = append(result, &entities.RoomMessageEventReactionEntity{
			Id:             *reactionUuidStr,
			RoomId:         *roomUuidStr,
			MessageEventId: *messageEventUuidStr,
			UserId:         *userUuidStr,
			Content:        content,
			DateCreate:     dateCreate,
		})
	}

	return result, nil
}
