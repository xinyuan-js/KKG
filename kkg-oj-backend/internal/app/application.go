package app

import (
	"yuoj-go-backend/internal/config"
	"yuoj-go-backend/internal/handler"
	"yuoj-go-backend/internal/infra/db"
	"yuoj-go-backend/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Application struct {
	cfg    *config.Config
	router *gin.Engine
}

func NewApplication(cfg *config.Config, logger *zap.Logger) (*Application, error) {
	gdb, err := db.NewMySQL(cfg)
	if err != nil {
		return nil, err
	}
	if err = migrate(gdb); err != nil {
		return nil, err
	}

	h := handler.New(gdb, service.NewUserService(gdb), cfg)
	startJudgeQueue(cfg, logger, h)

	router, err := NewHTTPServer(cfg, logger, h)
	if err != nil {
		return nil, err
	}
	return &Application{
		cfg:    cfg,
		router: router,
	}, nil
}

func (a *Application) Run() error {
	return a.router.Run(":" + a.cfg.Server.Port)
}
