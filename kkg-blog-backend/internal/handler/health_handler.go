package handler

import (
	"context"
	"time"

	"awesomeProject/internal/bootstrap"
	"awesomeProject/pkg/response"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	app *bootstrap.App
}

func NewHealthHandler(app *bootstrap.App) *HealthHandler {
	return &HealthHandler{app: app}
}

func (h *HealthHandler) Ping(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	sqlDB, err := h.app.DB.DB()
	if err != nil {
		response.ServerError(c, "db unavailable")
		return
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		response.ServerError(c, "db ping failed")
		return
	}
	if err := h.app.Redis.Ping(ctx).Err(); err != nil {
		response.ServerError(c, "redis ping failed")
		return
	}

	response.OK(c, gin.H{"status": "up"})
}
