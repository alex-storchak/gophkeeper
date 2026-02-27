package interceptor

import (
	"context"
	"log/slog"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/alex-storchak/gophkeeper/internal/models"
	"github.com/alex-storchak/gophkeeper/internal/server/ctxutil"
)

const bearerPrefix = "Bearer "

type TokenManager interface {
	Validate(token string) (models.UserID, error)
	Generate(userID models.UserID) (string, error)
}

type AuthInterceptor struct {
	tokenManager TokenManager
	logger       *slog.Logger
}

func NewAuthInterceptor(tm TokenManager, l *slog.Logger) *AuthInterceptor {
	return &AuthInterceptor{
		tokenManager: tm,
		logger:       l,
	}
}

func (i *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		userID, err := i.authenticate(ctx)
		if err != nil {
			return nil, err
		}

		ctx = ctxutil.WithUser(ctx, userID)

		resp, handlerErr := handler(ctx, req)

		newToken, err := i.tokenManager.Generate(userID)
		if err != nil {
			i.logger.Error("failed to refresh token", slog.Any("err", err))
		} else {
			md := metadata.Pairs("x-refreshed-token", newToken)
			if sendErr := grpc.SetHeader(ctx, md); sendErr != nil {
				i.logger.Error("failed to send refreshed token header", slog.Any("err", sendErr))
			}
		}

		return resp, handlerErr
	}
}

func (i *AuthInterceptor) authenticate(ctx context.Context) (models.UserID, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return 0, status.Error(codes.Unauthenticated, "missing metadata")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return 0, status.Error(codes.Unauthenticated, "missing authorization token")
	}

	token := values[0]
	if strings.HasPrefix(token, bearerPrefix) {
		token = strings.TrimPrefix(token, bearerPrefix)
	}

	userID, err := i.tokenManager.Validate(token)
	if err != nil {
		i.logger.Warn("invalid token", slog.Any("err", err))
		return 0, status.Error(codes.Unauthenticated, "invalid or expired token")
	}

	return userID, nil
}
