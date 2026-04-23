package user

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
	ErrUserNotFound = errors.New("user not found")
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) Create(ctx context.Context, email, name, avatar, provider string) (*models.User, error) {
	user := &models.User{
		ID:        uuid.New().String(),
		Email:     email,
		Name:      name,
		Avatar:    avatar,
		Provider:  provider,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO users (id, email, name, avatar, provider, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (email) DO UPDATE SET
			name = EXCLUDED.name,
			avatar = EXCLUDED.avatar,
			updated_at = EXCLUDED.updated_at
		RETURNING id, created_at
	`, user.ID, user.Email, user.Name, user.Avatar, user.Provider, user.CreatedAt, user.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, name, avatar, provider, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(&user.ID, &user.Email, &user.Name, &user.Avatar, &user.Provider, &user.CreatedAt, &user.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *Service) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, name, avatar, provider, created_at, updated_at
		FROM users WHERE email = $1
	`, email).Scan(&user.ID, &user.Email, &user.Name, &user.Avatar, &user.Provider, &user.CreatedAt, &user.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *Service) Update(ctx context.Context, user *models.User) error {
	user.UpdatedAt = time.Now()
	_, err := s.pool.Exec(ctx, `
		UPDATE users SET name = $2, avatar = $3, updated_at = $4
		WHERE id = $1
	`, user.ID, user.Name, user.Avatar, user.UpdatedAt)

	return err
}

func (s *Service) Delete(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}
