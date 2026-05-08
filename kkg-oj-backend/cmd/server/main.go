package main

import (
	"context"
	"log"
	"yuoj-go-backend/internal/app"
	"yuoj-go-backend/internal/config"
	"yuoj-go-backend/internal/handler"
	"yuoj-go-backend/internal/infra/db"
	"yuoj-go-backend/internal/infra/mq"
	"yuoj-go-backend/internal/middleware"
	"yuoj-go-backend/internal/model/entity"
	"yuoj-go-backend/internal/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	redisStore "github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config error: %v", err)
	}
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	gdb, err := db.NewMySQL(cfg)
	if err != nil {
		log.Fatalf("init mysql error: %v", err)
	}
	if err = gdb.AutoMigrate(
		&entity.User{},
		&entity.Question{},
		&entity.QuestionSubmit{},
		&entity.QuestionSolutionPost{},
		&entity.AgentSolutionTask{},
	); err != nil {
		log.Fatalf("auto migrate error: %v", err)
	}

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

	var store sessions.Store
	store, err = redisStore.NewStore(10, "tcp", cfg.Redis.Addr, "", cfg.Redis.Password, []byte("yuoj-go-secret"))
	if err != nil {
		logger.Warn("init redis session failed, fallback to cookie session", zap.Error(err))
		store = cookie.NewStore([]byte("yuoj-go-secret"))
	}
	r.Use(sessions.Sessions("yuoj_session", store))

	h := handler.New(gdb, service.NewUserService(gdb), cfg)
	if cfg.RabbitMQ.Enabled {
		judgeQueue, mqErr := mq.NewJudgeQueue(cfg.RabbitMQ.URL, cfg.RabbitMQ.JudgeQueue)
		if mqErr != nil {
			logger.Warn("init rabbitmq failed, fallback local async judge", zap.Error(mqErr))
		} else {
			h.SetJudgeSubmitter(judgeQueue)
			if consumeErr := judgeQueue.Consume(context.Background(), h.ConsumeJudge); consumeErr != nil {
				logger.Warn("consume judge queue failed, fallback local async judge", zap.Error(consumeErr))
				_ = judgeQueue.Close()
			} else {
				logger.Info("judge queue consumer started", zap.String("queue", cfg.RabbitMQ.JudgeQueue))
			}
		}
	}
	app.RegisterRoutes(r, h)

	if err := r.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("run server error: %v", err)
	}
}
