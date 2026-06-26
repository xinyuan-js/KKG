package app

import (
	"yuoj-go-backend/internal/config"
	"yuoj-go-backend/internal/handler"
	"yuoj-go-backend/internal/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	redisStore "github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func NewHTTPServer(cfg *config.Config, logger *zap.Logger, h *handler.Handler) (*gin.Engine, error) {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://127.0.0.1:3001",
			"http://localhost:3001",
			"http://127.0.0.1:5173",
			"http://localhost:5173",
			"http://127.0.0.1:8101",
			"http://localhost:8101",
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "auth"},
		ExposeHeaders:    []string{"Content-Length", "Set-Cookie"},
		AllowCredentials: true,
	}))
	r.Use(middleware.Recovery())
	r.Use(middleware.RequestLogger(logger))

	store, err := redisStore.NewStore(10, "tcp", cfg.Redis.Addr, "", cfg.Redis.Password, []byte("yuoj-go-secret"))
	if err != nil {
		logger.Warn("init redis session failed, fallback to cookie session", zap.Error(err))
		store = cookie.NewStore([]byte("yuoj-go-secret"))
	}
	r.Use(sessions.Sessions("yuoj_session", store))

	RegisterRoutes(r, h)
	return r, nil
}
