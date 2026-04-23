package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type PasswordService struct {
	pool *pgxpool.Pool
}

func NewPasswordService(pool *pgxpool.Pool) *PasswordService {
	return &PasswordService{pool: pool}
}

func (s *PasswordService) Register(ctx context.Context, email, password, name string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	userID := uuid.New().String()
	now := time.Now()

	_, err = s.pool.Exec(ctx, `
		INSERT INTO users (id, email, name, avatar, provider, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (email) DO NOTHING
	`, userID, email, name, "", "password", now, now)

	if err != nil {
		return "", err
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO user_passwords (user_id, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
	`, userID, string(hashedPassword), now, now)

	if err != nil {
		return "", err
	}

	return userID, nil
}

func (s *PasswordService) Login(ctx context.Context, email, password string) (string, error) {
	var userID string
	var passwordHash string

	err := s.pool.QueryRow(ctx, `
		SELECT u.id, up.password_hash
		FROM users u
		JOIN user_passwords up ON u.id = up.user_id
		WHERE u.email = $1 AND u.provider = 'password'
	`, email).Scan(&userID, &passwordHash)

	if err != nil {
		return "", ErrInvalidCredentials
	}

	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
		return "", ErrInvalidCredentials
	}

	return userID, nil
}

func (s *PasswordService) UserExists(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)
	`, email).Scan(&exists)
	return exists, err
}
