package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	sq "github.com/Masterminds/squirrel"
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

var dataEntryColumns = []string{
	"de.id",
	"dt.code",
	"de.title",
	"de.encrypted_data",
	"de.salt",
	"de.metadata",
	"de.created_at",
}
var dataEntrySummaryColumns = []string{"de.id", "dt.code", "de.title", "de.metadata", "de.created_at"}

func baseDataSelect(columns []string) sq.SelectBuilder {
	return psql.
		Select(columns...).
		From("data_entries de").
		Join("data_types dt ON dt.id = de.data_type_id")
}

func (r *PgDataRepo) GetByID(
	ctx context.Context,
	userID models.UserID,
	entryID models.DataEntryID,
) (*models.DataEntry, error) {
	query, args, err := baseDataSelect(dataEntryColumns).
		Where(sq.Eq{"de.id": int64(entryID)}).
		Where(sq.Eq{"de.user_id": int64(userID)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("building get-by-id query: %w", err)
	}
	return r.getOne(ctx, query, args...)
}

func (r *PgDataRepo) GetByTitle(
	ctx context.Context,
	userID models.UserID,
	title string,
) (*models.DataEntry, error) {
	query, args, err := baseDataSelect(dataEntryColumns).
		Where(sq.Eq{"de.title": title}).
		Where(sq.Eq{"de.user_id": int64(userID)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("building get-by-title query: %w", err)
	}
	return r.getOne(ctx, query, args...)
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
	qb := baseDataSelect(dataEntrySummaryColumns).
		Where(sq.Eq{"de.user_id": int64(userID)})

	if dataType != "" {
		qb = qb.Where(sq.Eq{"dt.code": dataType})
	}

	query, args, err := qb.
		OrderBy("de.created_at DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("building list query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
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
	query, args, err := psql.
		Delete("data_entries").
		Where(sq.Eq{"id": int64(entryID)}).
		Where(sq.Eq{"user_id": int64(userID)}).
		ToSql()
	if err != nil {
		return fmt.Errorf("building delete-by-id query: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("deleting data entry by id: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrDataNotFound
	}
	return nil
}

func (r *PgDataRepo) DeleteByTitle(ctx context.Context, userID models.UserID, title string) error {
	query, args, err := psql.
		Delete("data_entries").
		Where(sq.Eq{"title": title}).
		Where(sq.Eq{"user_id": int64(userID)}).
		ToSql()
	if err != nil {
		return fmt.Errorf("building delete-by-title query: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
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

	query, args, err := baseDataSelect(dataEntrySummaryColumns).
		Where(sq.Eq{"de.title": title}).
		Where(sq.Eq{"de.user_id": int64(userID)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("building summary-by-title query: %w", err)
	}

	err = r.pool.QueryRow(ctx, query, args...).
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
