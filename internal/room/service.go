package room

import (
	"context"
	"errors"
	"time"

	"voicechat/pkg/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRoomNotFound = errors.New("room not found")
	ErrRoomFull     = errors.New("room is full")
	ErrUnauthorized = errors.New("unauthorized to join room")
)

const MaxRoomMembers = 10

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) Create(ctx context.Context, name, ownerID string) (*models.Room, error) {
	room := &models.Room{
		ID:        uuid.New().String(),
		Name:      name,
		OwnerID:   ownerID,
		CreatedAt: time.Now(),
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO rooms (id, name, owner_id, created_at)
		VALUES ($1, $2, $3, $4)
	`, room.ID, room.Name, room.OwnerID, room.CreatedAt)

	if err != nil {
		return nil, err
	}

	return room, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*models.Room, error) {
	var room models.Room
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, owner_id, created_at, closed_at
		FROM rooms WHERE id = $1 AND closed_at IS NULL
	`, id).Scan(&room.ID, &room.Name, &room.OwnerID, &room.CreatedAt, &room.ClosedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRoomNotFound
	}
	if err != nil {
		return nil, err
	}

	return &room, nil
}

func (s *Service) List(ctx context.Context, limit, offset int) ([]*models.Room, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, owner_id, created_at, closed_at
		FROM rooms WHERE closed_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []*models.Room
	for rows.Next() {
		var room models.Room
		if err := rows.Scan(&room.ID, &room.Name, &room.OwnerID, &room.CreatedAt, &room.ClosedAt); err != nil {
			return nil, err
		}
		rooms = append(rooms, &room)
	}

	return rooms, nil
}

func (s *Service) Close(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE rooms SET closed_at = $2 WHERE id = $1
	`, id, time.Now())
	return err
}

func (s *Service) Join(ctx context.Context, roomID, userID string) (*models.RoomMember, error) {
	count, err := s.GetMemberCount(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if count >= MaxRoomMembers {
		return nil, ErrRoomFull
	}

	member := &models.RoomMember{
		ID:       uuid.New().String(),
		RoomID:   roomID,
		UserID:   userID,
		JoinedAt: time.Now(),
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO room_members (id, room_id, user_id, joined_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (room_id, user_id) DO UPDATE SET joined_at = EXCLUDED.joined_at, left_at = NULL
	`, member.ID, member.RoomID, member.UserID, member.JoinedAt)

	if err != nil {
		return nil, err
	}

	return member, nil
}

func (s *Service) Leave(ctx context.Context, roomID, userID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE room_members SET left_at = $3 WHERE room_id = $1 AND user_id = $2 AND left_at IS NULL
	`, roomID, userID, time.Now())
	return err
}

func (s *Service) GetMembers(ctx context.Context, roomID string) ([]*models.RoomMember, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, room_id, user_id, joined_at, left_at
		FROM room_members WHERE room_id = $1 AND left_at IS NULL
	`, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*models.RoomMember
	for rows.Next() {
		var m models.RoomMember
		if err := rows.Scan(&m.ID, &m.RoomID, &m.UserID, &m.JoinedAt, &m.LeftAt); err != nil {
			return nil, err
		}
		members = append(members, &m)
	}

	return members, nil
}

func (s *Service) GetMemberCount(ctx context.Context, roomID string) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM room_members WHERE room_id = $1 AND left_at IS NULL
	`, roomID).Scan(&count)
	return count, err
}

func (s *Service) IsOwner(ctx context.Context, roomID, userID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM rooms WHERE id = $1 AND owner_id = $2)
	`, roomID, userID).Scan(&exists)
	return exists, err
}
