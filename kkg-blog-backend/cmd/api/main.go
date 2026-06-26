package main

import (
	"log"

	"awesomeProject/internal/bootstrap"
	"awesomeProject/internal/router"
)

func main() {
	app, err := bootstrap.NewApp()
	if err != nil {
		log.Fatalf("bootstrap app failed: %v", err)
	}

	r := router.New(app)
	if err := r.Run(":" + app.Config.ServerPort); err != nil {
		log.Fatalf("start server failed: %v", err)
	}
}
