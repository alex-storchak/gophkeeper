package grpcserver

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/selector"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pb "github.com/alex-storchak/gophkeeper/gen/proto/gophkeeper/v1"
	"github.com/alex-storchak/gophkeeper/internal/server/config"
	"github.com/alex-storchak/gophkeeper/internal/server/interceptor"
)

const maxMsgSize = 64 * 1024 * 1024 // 64 MB

type ServerDeps struct {
	Config      *config.Config
	Logger      *slog.Logger
	AuthService AuthServicer
	DataService DataServicer
	JWTManager  interceptor.TokenManager
}

func Serve(deps *ServerDeps) (*grpc.Server, error) {
	logger := deps.Logger
	cfg := deps.Config

	server, err := newGrpcServer(deps)
	if err != nil {
		return nil, fmt.Errorf("create grpc server: %w", err)
	}

	authHandler := NewAuthHandler(deps.AuthService, deps.Logger)
	dataHandler := NewDataHandler(deps.DataService, deps.Logger)

	pb.RegisterAuthServiceServer(server, authHandler)
	pb.RegisterDataServiceServer(server, dataHandler)

	errCh := make(chan error, 1)
	go func() {
		logger.Info("starting server", slog.String("server address", cfg.GRPCAddress))
		lis, err := net.Listen("tcp", cfg.GRPCAddress)
		if err != nil {
			errCh <- fmt.Errorf("listen grpc address: %w", err)
			return
		}
		if err := server.Serve(lis); err != nil {
			errCh <- fmt.Errorf("start to serve: %w", err)
		}
	}()

	// Проверяем, успешно ли запустился сервер
	select {
	case err := <-errCh:
		return nil, fmt.Errorf("start server: %w", err)
	case <-time.After(time.Second):
		return server, nil
	}
}

func newGrpcServer(deps *ServerDeps) (*grpc.Server, error) {
	authInterceptor := interceptor.NewAuthInterceptor(deps.JWTManager, deps.Logger)

	matchFunc := func(ctx context.Context, callMeta interceptors.CallMeta) bool {
		return callMeta.Service == pb.DataService_ServiceDesc.ServiceName
	}

	creds, err := credentials.NewServerTLSFromFile(deps.Config.TLS.CertFile, deps.Config.TLS.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("creating server credentials: %w", err)
	}

	srv := grpc.NewServer(
		grpc.Creds(creds),
		grpc.MaxRecvMsgSize(maxMsgSize),
		grpc.MaxSendMsgSize(maxMsgSize),
		grpc.ChainUnaryInterceptor(
			selector.UnaryServerInterceptor(authInterceptor.Unary(), selector.MatchFunc(matchFunc)),
		),
	)

	return srv, nil
}
