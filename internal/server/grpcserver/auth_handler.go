package grpcserver

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/alex-storchak/gophkeeper/gen/proto/gophkeeper/v1"
	"github.com/alex-storchak/gophkeeper/internal/server/repository"
	"github.com/alex-storchak/gophkeeper/internal/server/service"
)

type AuthServicer interface {
	Register(ctx context.Context, username, password string) (string, error)
	Login(ctx context.Context, username, password string) (string, error)
}

type AuthHandler struct {
	pb.UnimplementedAuthServiceServer
	authService AuthServicer
	logger      *slog.Logger
}

func NewAuthHandler(authService AuthServicer, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
	}
}

func (h *AuthHandler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	if req.GetUsername() == "" || req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "username and password are required")
	}

	token, err := h.authService.Register(ctx, req.GetUsername(), req.GetPassword())
	if errors.Is(err, repository.ErrUserAlreadyExists) {
		h.logger.Error("registration failed", slog.Any("err", err))
		return nil, status.Error(codes.AlreadyExists, err.Error())
	} else if err != nil {
		h.logger.Error("failed to register user", slog.Any("err", err))
		return nil, status.Error(codes.Internal, "internal error")
	}

	res := pb.RegisterResponse_builder{
		Token: token,
	}.Build()

	return res, nil
}

func (h *AuthHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req.GetUsername() == "" || req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "username and password are required")
	}

	token, err := h.authService.Login(ctx, req.GetUsername(), req.GetPassword())
	if errors.Is(err, service.ErrInvalidCredentials) {
		h.logger.Warn("login failed",
			slog.String("username", req.GetUsername()),
			slog.Any("err", err),
		)
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	} else if err != nil {
		h.logger.Error("failed to login user", slog.Any("err", err))
		return nil, status.Error(codes.Internal, "internal error")
	}

	res := pb.LoginResponse_builder{
		Token: token,
	}.Build()
	return res, nil
}
