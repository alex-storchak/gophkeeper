package factory

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alex-storchak/gophkeeper/internal/server/config"
	"github.com/alex-storchak/gophkeeper/internal/server/db"
	"github.com/alex-storchak/gophkeeper/internal/server/repository"
	"github.com/alex-storchak/gophkeeper/internal/server/service"
)

type Repo interface {
	MakeData() service.DataRepo
	MakeUser() service.UserRepo
}

type PgRepo struct {
	dbPool *pgxpool.Pool
	logger *slog.Logger
}

func NewPgRepo(ctx context.Context, cfg config.DB, l *slog.Logger) (*PgRepo, error) {
	pgDB, err := db.NewPgDB(ctx, cfg, l)
	if err != nil {
		return nil, fmt.Errorf("create pg db pool: %w", err)
	}
	return &PgRepo{
		dbPool: pgDB,
		logger: l,
	}, nil
}

func (p *PgRepo) MakeData() service.DataRepo {
	return repository.NewPgDataRepository(p.dbPool, p.logger)
}

func (p *PgRepo) MakeUser() service.UserRepo {
	return repository.NewPgUserRepository(p.dbPool, p.logger)
}
