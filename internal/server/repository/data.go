package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alex-storchak/gophkeeper/internal/models"
)

var (
	ErrDataNotFound  = errors.New("data entry not found")
	ErrTitleConflict = errors.New("data entry with this title already exists")
)

type PgDataRepo struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewPgDataRepository(p *pgxpool.Pool, l *slog.Logger) *PgDataRepo {
	return &PgDataRepo{
		pool:   p,
		logger: l,
	}
}

func (r *PgDataRepo) Close() {
	r.pool.Close()
}

func (r *PgDataRepo) Create(ctx context.Context, entry *models.DataEntry) (models.DataEntryID, error) {
	var id int64
	q := `
		INSERT INTO data_entries (user_id, data_type_id, title, encrypted_data, salt, metadata)
		SELECT $1, dt.id, $3, $4, $5, $6
		FROM data_types dt
		WHERE dt.code = $2
		RETURNING data_entries.id
	`

	err := r.pool.QueryRow(
		ctx,
		q,
		int64(entry.UserID),
		entry.DataType,
		entry.Title,
		entry.EncryptedData,
		entry.Salt,
		entry.Metadata,
	).Scan(&id)
	if err != nil {
		const uniqueViolationCode = "23505"
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return 0, ErrTitleConflict
		}
		return 0, fmt.Errorf("inserting data entry: %w", err)
	}
	return models.DataEntryID(id), nil
}

func (r *PgDataRepo) GetByID(
	ctx context.Context,
	userID models.UserID,
	entryID models.DataEntryID,
) (*models.DataEntry, error) {
	q := `
		SELECT de.id, dt.code, de.title, de.encrypted_data, de.salt, de.metadata, de.created_at
		FROM data_entries de
		JOIN data_types dt ON dt.id = de.data_type_id
		WHERE de.id = $1
		AND de.user_id = $2
	`
	return r.getOne(ctx, q, int64(entryID), int64(userID))
}

func (r *PgDataRepo) GetByTitle(
	ctx context.Context,
	userID models.UserID,
	title string,
) (*models.DataEntry, error) {
	q := `
		SELECT de.id, dt.code, de.title, de.encrypted_data, de.salt, de.metadata, de.created_at
		FROM data_entries de
		JOIN data_types dt ON dt.id = de.data_type_id
		WHERE de.title = $1
		AND de.user_id = $2
	`
	return r.getOne(ctx, q, title, int64(userID))
}

func (r *PgDataRepo) getOne(ctx context.Context, query string, args ...any) (*models.DataEntry, error) {
	var entry models.DataEntry
	var id int64

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&id,
		&entry.DataType,
		&entry.Title,
		&entry.EncryptedData,
		&entry.Salt,
		&entry.Metadata,
		&entry.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDataNotFound
		}
		return nil, fmt.Errorf("querying data entry: %w", err)
	}

	entry.ID = models.DataEntryID(id)
	return &entry, nil
}

func (r *PgDataRepo) List(ctx context.Context, userID models.UserID, dataType string) ([]models.DataEntrySummary, error) {
	q := `
		SELECT de.id, dt.code, de.title, de.metadata, de.created_at
		FROM data_entries de
		JOIN data_types dt ON dt.id = de.data_type_id
		WHERE de.user_id = $1
	`
	args := []any{int64(userID)}

	if dataType != "" {
		q += ` AND dt.code = $2`
		args = append(args, dataType)
	}

	q += ` ORDER BY de.created_at DESC`

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("querying data entries list: %w", err)
	}
	defer rows.Close()

	var entries []models.DataEntrySummary
	for rows.Next() {
		var s models.DataEntrySummary
		var id int64
		if err := rows.Scan(&id, &s.DataType, &s.Title, &s.Metadata, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning data entry row: %w", err)
		}
		s.ID = models.DataEntryID(id)
		entries = append(entries, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating data entry rows: %w", err)
	}

	return entries, nil
}

func (r *PgDataRepo) DeleteByID(ctx context.Context, userID models.UserID, entryID models.DataEntryID) error {
	q := `DELETE FROM data_entries WHERE id = $1 AND user_id = $2`
	tag, err := r.pool.Exec(ctx, q, int64(entryID), int64(userID))
	if err != nil {
		return fmt.Errorf("deleting data entry by id: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrDataNotFound
	}
	return nil
}

func (r *PgDataRepo) DeleteByTitle(ctx context.Context, userID models.UserID, title string) error {
	q := `DELETE FROM data_entries WHERE title = $1 AND user_id = $2`
	tag, err := r.pool.Exec(ctx, q, title, int64(userID))
	if err != nil {
		return fmt.Errorf("deleting data entry by title: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrDataNotFound
	}
	return nil
}

func (r *PgDataRepo) GetSummaryByTitle(
	ctx context.Context,
	userID models.UserID,
	title string,
) (*models.DataEntrySummary, error) {
	var s models.DataEntrySummary
	var id int64
	q := `
		SELECT de.id, dt.code, de.title, de.metadata, de.created_at
		FROM data_entries de
		JOIN data_types dt ON dt.id = de.data_type_id
		WHERE de.title = $1 
		AND de.user_id = $2
	`
	err := r.pool.QueryRow(ctx, q, title, int64(userID)).
		Scan(&id, &s.DataType, &s.Title, &s.Metadata, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDataNotFound
		}
		return nil, fmt.Errorf("querying data entry summary: %w", err)
	}
	s.ID = models.DataEntryID(id)
	return &s, nil
}
