package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/alex-storchak/gophkeeper/internal/server/config"
	"github.com/alex-storchak/gophkeeper/internal/server/grpcserver"
	"github.com/alex-storchak/gophkeeper/internal/server/jwt"
	"github.com/alex-storchak/gophkeeper/internal/server/logger"
	"github.com/alex-storchak/gophkeeper/internal/server/repository/factory"
	"github.com/alex-storchak/gophkeeper/internal/server/service"
)

func Run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	sl, closeLog, err := initLogger(cfg)
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer func() {
		_ = closeLog()
	}()

	pgFactory, err := factory.NewPgRepo(ctx, cfg.DB, sl)
	if err != nil {
		return fmt.Errorf("create pg repo factory: %w", err)
	}
	userRepo := pgFactory.MakeUser()
	dataRepo := pgFactory.MakeData()

	jwtManager := jwt.NewManager(&cfg.Auth)

	authService := service.NewAuthService(userRepo, jwtManager, sl)
	defer authService.Close()
	dataService := service.NewDataService(dataRepo, sl)
	defer dataService.Close()

	grpcServer, err := grpcserver.Serve(&grpcserver.ServerDeps{
		Config:      cfg,
		Logger:      sl,
		AuthService: authService,
		DataService: dataService,
		JWTManager:  jwtManager,
	})

	<-ctx.Done()

	// shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownWaitSecsDuration)
	defer cancel()

	// shutdown grpc server
	grpcStopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(grpcStopped)
	}()

	select {
	case <-grpcStopped:
		sl.Info("grpc server stopped gracefully")
	case <-shutdownCtx.Done():
		sl.Warn("grpc shutdown forced by timeout")
		grpcServer.Stop()
	}

	return nil
}

func initLogger(cfg *config.Config) (*slog.Logger, func() error, error) {
	sl, closeLog, err := logger.New(cfg.Log)
	if err != nil {
		return nil, nil, fmt.Errorf("new logger; config: %v; error: %w", cfg.Log, err)
	}
	sl.Info("logger initialized")
	return sl, closeLog, nil
}
