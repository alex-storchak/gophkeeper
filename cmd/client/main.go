package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/alex-storchak/gophkeeper/internal/client/app"
	"github.com/alex-storchak/gophkeeper/internal/client/cmd"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
)

func main() {
	cmd.SetBuildInfo(buildVersion, buildDate)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer cancel()

	if err := app.Run(ctx); err != nil {
		os.Exit(1)
	}
}
