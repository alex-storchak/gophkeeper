package grpcserver

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/alex-storchak/gophkeeper/gen/proto/gophkeeper/v1"
	"github.com/alex-storchak/gophkeeper/internal/models"
	"github.com/alex-storchak/gophkeeper/internal/server/ctxutil"
	"github.com/alex-storchak/gophkeeper/internal/server/repository"
	"github.com/alex-storchak/gophkeeper/internal/server/service"
)

type DataServicer interface {
	Add(ctx context.Context, entry *models.DataEntry) (models.DataEntryID, error)
	Get(ctx context.Context, userID models.UserID, entryID models.DataEntryID, title string) (*models.DataEntry, error)
	List(ctx context.Context, userID models.UserID, dataType string) ([]models.DataEntrySummary, error)
	Delete(ctx context.Context, userID models.UserID, entryID models.DataEntryID, title string) error
	GetConflictInfo(ctx context.Context, userID models.UserID, title string) (*models.DataEntrySummary, error)
}

type DataHandler struct {
	pb.UnimplementedDataServiceServer
	dataService DataServicer
	logger      *slog.Logger
}

func NewDataHandler(dataService DataServicer, logger *slog.Logger) *DataHandler {
	return &DataHandler{
		dataService: dataService,
		logger:      logger,
	}
}

func (h *DataHandler) Add(ctx context.Context, req *pb.AddRequest) (*pb.AddResponse, error) {
	userID, err := ctxutil.GetCtxUserID(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to identify user")
	}
	if req.GetTitle() == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}
	if len(req.GetEncryptedData()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "data is required")
	}

	entry := &models.DataEntry{
		UserID:        userID,
		DataType:      req.GetDataType(),
		Title:         req.GetTitle(),
		EncryptedData: req.GetEncryptedData(),
		Salt:          req.GetSalt(),
		Metadata:      req.GetMetadata(),
	}

	id, err := h.dataService.Add(ctx, entry)
	if err != nil {
		if errors.Is(err, repository.ErrTitleConflict) {
			summary, conflictErr := h.dataService.GetConflictInfo(ctx, userID, req.GetTitle())
			if conflictErr != nil {
				h.logger.Error("failed to get conflict info", slog.Any("err", conflictErr))
				return nil, status.Error(codes.AlreadyExists, "entry with this title already exists")
			}

			conflictInfo := pb.ConflictInfo_builder{
				Id:        int64(summary.ID),
				DataType:  summary.DataType,
				Title:     summary.Title,
				Metadata:  summary.Metadata,
				CreatedAt: timestamppb.New(summary.CreatedAt),
			}.Build()

			st, stErr := status.New(codes.AlreadyExists, "entry with this title already exists").
				WithDetails(conflictInfo)
			if stErr != nil {
				h.logger.Error("failed to attach conflict details", slog.Any("err", stErr))
				return nil, status.Error(codes.AlreadyExists, "entry with this title already exists")
			}
			return nil, st.Err()
		}

		h.logger.Error("failed to add data entry", slog.Any("err", err))
		return nil, status.Error(codes.Internal, "failed to add data entry")
	}

	res := pb.AddResponse_builder{
		Id: int64(id),
	}.Build()
	return res, nil
}

func (h *DataHandler) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	userID, err := ctxutil.GetCtxUserID(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to identify user")
	}

	entry, err := h.dataService.Get(ctx, userID, models.DataEntryID(req.GetId()), req.GetTitle())
	if err != nil {
		if errors.Is(err, service.ErrEmptyIDAndTitle) {
			return nil, status.Error(codes.InvalidArgument, "either id or title must be provided")
		}
		if errors.Is(err, repository.ErrDataNotFound) {
			return nil, status.Error(codes.NotFound, "data entry not found")
		}
		return nil, status.Error(codes.Internal, "failed to get data entry")
	}

	res := pb.GetResponse_builder{
		Id:            int64(entry.ID),
		DataType:      entry.DataType,
		Title:         entry.Title,
		EncryptedData: entry.EncryptedData,
		Salt:          entry.Salt,
		Metadata:      entry.Metadata,
		CreatedAt:     timestamppb.New(entry.CreatedAt),
	}.Build()
	return res, nil
}

func (h *DataHandler) List(ctx context.Context, req *pb.ListRequest) (*pb.ListResponse, error) {
	userID, err := ctxutil.GetCtxUserID(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to identify user")
	}

	entries, err := h.dataService.List(ctx, userID, req.GetDataType())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list data entries")
	}

	summaries := make([]*pb.DataEntrySummary, 0, len(entries))
	for _, e := range entries {
		summary := pb.DataEntrySummary_builder{
			Id:        int64(e.ID),
			DataType:  e.DataType,
			Title:     e.Title,
			Metadata:  e.Metadata,
			CreatedAt: timestamppb.New(e.CreatedAt),
		}.Build()
		summaries = append(summaries, summary)
	}

	resp := pb.ListResponse_builder{
		Entries: summaries,
	}.Build()

	return resp, nil
}

func (h *DataHandler) Delete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	userID, err := ctxutil.GetCtxUserID(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to identify user")
	}

	err = h.dataService.Delete(ctx, userID, models.DataEntryID(req.GetId()), req.GetTitle())
	if err != nil {
		if errors.Is(err, service.ErrEmptyIDAndTitle) {
			return nil, status.Error(codes.InvalidArgument, "either id or title must be provided")
		}
		if errors.Is(err, repository.ErrDataNotFound) {
			return nil, status.Error(codes.NotFound, "data entry not found")
		}
		return nil, status.Error(codes.Internal, "failed to delete data entry")
	}

	return &pb.DeleteResponse{}, nil
}
