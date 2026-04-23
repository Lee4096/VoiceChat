package postgres

import (
	"context"
	"fmt"
	"time"

	"voicechat/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, cfg config.DatabaseConfig) (*DB, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.DBName,
		cfg.SSLMode,
	)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{pool: pool}, nil
}

func (db *DB) Close() {
	if db.pool != nil {
		db.pool.Close()
	}
}

func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}

func (db *DB) InitSchema(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id VARCHAR(36) PRIMARY KEY,
		email VARCHAR(255) UNIQUE NOT NULL,
		name VARCHAR(255) NOT NULL,
		avatar TEXT,
		provider VARCHAR(50) NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS user_passwords (
		user_id VARCHAR(36) PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
		password_hash TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS rooms (
		id VARCHAR(36) PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		owner_id VARCHAR(36) NOT NULL REFERENCES users(id),
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		closed_at TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS room_members (
		id VARCHAR(36) PRIMARY KEY,
		room_id VARCHAR(36) NOT NULL REFERENCES rooms(id),
		user_id VARCHAR(36) NOT NULL REFERENCES users(id),
		joined_at TIMESTAMP NOT NULL DEFAULT NOW(),
		left_at TIMESTAMP,
		UNIQUE(room_id, user_id)
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id VARCHAR(36) PRIMARY KEY,
		user_id VARCHAR(36) NOT NULL REFERENCES users(id),
		token TEXT UNIQUE NOT NULL,
		expires_at TIMESTAMP NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_rooms_owner ON rooms(owner_id);
	CREATE INDEX IF NOT EXISTS idx_room_members_room ON room_members(room_id);
	CREATE INDEX IF NOT EXISTS idx_room_members_user ON room_members(user_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
	CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
	`

	_, err := db.pool.Exec(ctx, schema)
	return err
}
