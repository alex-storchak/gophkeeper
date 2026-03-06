package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/alex-storchak/gophkeeper/internal/server/app"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer cancel()

	if err := app.Run(ctx); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
