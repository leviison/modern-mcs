package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"myconnectionsvr/modern-mcs/internal/app"
	"myconnectionsvr/modern-mcs/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	a, err := app.New(cfg)
	if err != nil {
		log.Fatalf("create app: %v", err)
	}

	if err := a.Run(ctx); err != nil {
		log.Fatalf("run app: %v", err)
	}
}
