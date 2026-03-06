package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/alex-storchak/gophkeeper/internal/client/cmd"
	"github.com/alex-storchak/gophkeeper/internal/client/config"
	"github.com/alex-storchak/gophkeeper/internal/client/grpcclient"
	"github.com/alex-storchak/gophkeeper/internal/client/logger"
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
	defer closeLog()

	client, err := grpcclient.New(cfg, sl)
	if err != nil {
		return fmt.Errorf("init grpc client: %w", err)
	}
	defer func() {
		closeErr := client.Close()
		sl.Debug("grpc client closed")
		if closeErr != nil {
			sl.Error("failed to close grpc client", "error", closeErr)
		}
	}()

	cmdDeps := cmd.Deps{
		Config:     cfg,
		Logger:     sl,
		UserClient: client,
		DataClient: client,
	}

	rootCmd := cmd.NewRootCmd(&cmdDeps)
	err = rootCmd.ExecuteContext(ctx)
	if err != nil {
		sl.Error("Failed to execute command", slog.Any("err", err))
		fmt.Printf("\nError: %v\n", err.Error())
	}
	return err
}

func initLogger(cfg *config.Config) (*slog.Logger, func() error, error) {
	sl, closeLog, err := logger.New(cfg.Log)
	if err != nil {
		return nil, nil, fmt.Errorf("new logger; config: %v; error: %w", cfg.Log, err)
	}
	sl.Info("logger initialized")
	return sl, closeLog, nil
}
