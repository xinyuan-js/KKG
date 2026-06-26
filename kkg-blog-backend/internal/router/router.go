package router

import (
	"awesomeProject/internal/bootstrap"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func New(app *bootstrap.App) *gin.Engine {
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://127.0.0.1:3001", "http://localhost:3001"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	deps := newDependencies(app)
	RegisterRoutes(r, app, deps)

	return r
}
