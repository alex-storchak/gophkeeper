package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alex-storchak/gophkeeper/internal/models"
)

var ErrUserNotFound = errors.New("user not found")
var ErrUserAlreadyExists = errors.New("user already exists")

type PgUserRepository struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewPgUserRepository(p *pgxpool.Pool, l *slog.Logger) *PgUserRepository {
	return &PgUserRepository{
		pool:   p,
		logger: l,
	}
}

func (r *PgUserRepository) Close() {
	r.pool.Close()
}

func (r *PgUserRepository) Create(ctx context.Context, username, passwordHash string) (models.UserID, error) {
	var id int64
	q := `
		INSERT INTO users (username, password_hash) 
		VALUES ($1, $2)
		ON CONFLICT (username) DO NOTHING
		RETURNING id
	`
	err := r.pool.QueryRow(ctx, q, username, passwordHash).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrUserAlreadyExists
		}
		return 0, fmt.Errorf("inserting user: %w", err)
	}

	return models.UserID(id), nil
}

func (r *PgUserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	var id int64

	q := `SELECT id, username, password_hash, created_at FROM users WHERE username = $1`
	err := r.pool.QueryRow(ctx, q, username).
		Scan(&id, &user.Username, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("querying user by username: %w", err)
	}

	user.ID = models.UserID(id)
	return &user, nil
}
