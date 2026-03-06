package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/alex-storchak/gophkeeper/internal/server/config"
)

func NewPgDB(ctx context.Context, cfg config.DB, l *slog.Logger) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("create pgx connection pool: %w", err)
	}

	if err = applyMigrations(&cfg, l); err != nil {
		return nil, fmt.Errorf("apply migrations: %w", err)
	}

	return pool, nil
}

func applyMigrations(cfg *config.DB, l *slog.Logger) error {
	mg, err := migrate.New(cfg.MigrationsPath, cfg.DSN)
	if err != nil {
		return fmt.Errorf("init database for migrations: %w", err)
	}
	err = mg.Up()
	if errors.Is(err, migrate.ErrNoChange) {
		l.Info("No new migrations to apply")
	} else if err != nil {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}
