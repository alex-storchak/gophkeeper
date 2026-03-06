package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/alex-storchak/gophkeeper/internal/models"
	"github.com/alex-storchak/gophkeeper/internal/server/repository"
)

var ErrEmptyIDAndTitle = errors.New("either id or title must be provided")

type DataRepo interface {
	Create(ctx context.Context, entry *models.DataEntry) (models.DataEntryID, error)
	GetByID(ctx context.Context, userID models.UserID, entryID models.DataEntryID) (*models.DataEntry, error)
	GetByTitle(ctx context.Context, userID models.UserID, title string) (*models.DataEntry, error)
	List(ctx context.Context, userID models.UserID, dataType string) ([]models.DataEntrySummary, error)
	DeleteByID(ctx context.Context, userID models.UserID, entryID models.DataEntryID) error
	DeleteByTitle(ctx context.Context, userID models.UserID, title string) error
	GetSummaryByTitle(ctx context.Context, userID models.UserID, title string) (*models.DataEntrySummary, error)
	Close()
}

type DataService struct {
	dataRepo DataRepo
	logger   *slog.Logger
}

func NewDataService(dataRepo DataRepo, logger *slog.Logger) *DataService {
	return &DataService{
		dataRepo: dataRepo,
		logger:   logger,
	}
}

func (s *DataService) Add(ctx context.Context, entry *models.DataEntry) (models.DataEntryID, error) {
	id, err := s.dataRepo.Create(ctx, entry)
	if errors.Is(err, repository.ErrTitleConflict) {
		return 0, err
	} else if err != nil {
		return 0, fmt.Errorf("creating data entry: %w; user_id: %d", err, int64(entry.UserID))
	}

	s.logger.Info("data entry created",
		slog.Int64("user_id", int64(entry.UserID)),
		slog.Int64("entry_id", int64(id)),
		slog.String("title", entry.Title),
	)
	return id, nil
}

func (s *DataService) Get(
	ctx context.Context,
	userID models.UserID,
	entryID models.DataEntryID,
	title string,
) (*models.DataEntry, error) {
	var entry *models.DataEntry
	var err error

	if entryID > 0 {
		entry, err = s.dataRepo.GetByID(ctx, userID, entryID)
	} else if title != "" {
		entry, err = s.dataRepo.GetByTitle(ctx, userID, title)
	} else {
		return nil, ErrEmptyIDAndTitle
	}

	if err != nil {
		if errors.Is(err, repository.ErrDataNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("getting data entry: %w", err)
	}

	entry.UserID = userID
	return entry, nil
}

func (s *DataService) List(
	ctx context.Context,
	userID models.UserID,
	dataType string,
) ([]models.DataEntrySummary, error) {
	entries, err := s.dataRepo.List(ctx, userID, dataType)
	if err != nil {
		return nil, fmt.Errorf("listing data entries: %w", err)
	}
	return entries, nil
}

func (s *DataService) Delete(
	ctx context.Context,
	userID models.UserID,
	entryID models.DataEntryID,
	title string,
) error {
	var err error

	if entryID > 0 {
		err = s.dataRepo.DeleteByID(ctx, userID, entryID)
	} else if title != "" {
		err = s.dataRepo.DeleteByTitle(ctx, userID, title)
	} else {
		return ErrEmptyIDAndTitle
	}

	if err != nil {
		if errors.Is(err, repository.ErrDataNotFound) {
			return err
		}
		return fmt.Errorf("deleting data entry: %w", err)
	}

	s.logger.Info("data entry deleted",
		slog.Int64("user_id", int64(userID)),
		slog.Int64("entry_id", int64(entryID)),
		slog.String("title", title),
	)
	return nil
}

func (s *DataService) GetConflictInfo(
	ctx context.Context,
	userID models.UserID,
	title string,
) (*models.DataEntrySummary, error) {
	summary, err := s.dataRepo.GetSummaryByTitle(ctx, userID, title)
	if err != nil {
		return nil, fmt.Errorf("getting conflict info: %w", err)
	}
	return summary, nil
}

func (s *DataService) Close() {
	s.dataRepo.Close()
}
