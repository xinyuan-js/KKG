package main

import (
	"log"

	"yuoj-go-backend/internal/app"
	"yuoj-go-backend/internal/config"

	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config error: %v", err)
	}
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	application, err := app.NewApplication(cfg, logger)
	if err != nil {
		log.Fatalf("init application error: %v", err)
	}

	if err := application.Run(); err != nil {
		log.Fatalf("run server error: %v", err)
	}
}
