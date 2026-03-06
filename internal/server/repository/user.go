package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alex-storchak/gophkeeper/internal/models"
)

var ErrUserNotFound = errors.New("user not found")
var ErrUserAlreadyExists = errors.New("user already exists")

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

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

	query, args, err := psql.
		Insert("users").
		Columns("username", "password_hash").
		Values(username, passwordHash).
		Suffix("ON CONFLICT (username) DO NOTHING").
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("building insert user query: %w", err)
	}

	err = r.pool.QueryRow(ctx, query, args...).Scan(&id)
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

	query, args, err := psql.
		Select("id", "username", "password_hash", "created_at").
		From("users").
		Where(sq.Eq{"username": username}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("building select user query: %w", err)
	}

	err = r.pool.QueryRow(ctx, query, args...).
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
