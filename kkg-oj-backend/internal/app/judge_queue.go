package app

import (
	"context"

	"yuoj-go-backend/internal/config"
	"yuoj-go-backend/internal/handler"
	"yuoj-go-backend/internal/infra/mq"

	"go.uber.org/zap"
)

func startJudgeQueue(cfg *config.Config, logger *zap.Logger, h *handler.Handler) {
	if !cfg.RabbitMQ.Enabled {
		return
	}

	judgeQueue, err := mq.NewJudgeQueue(cfg.RabbitMQ.URL, cfg.RabbitMQ.JudgeQueue)
	if err != nil {
		logger.Warn("init rabbitmq failed, fallback local async judge", zap.Error(err))
		return
	}
	judgeQueue.SetDeadLetterHandler(func(submitID int64, reason string, retryCount int32) {
		h.MarkSubmitSystemError(submitID, reason)
		logger.Error("judge submit moved to dlq",
			zap.Int64("submitId", submitID),
			zap.Int32("retryCount", retryCount),
			zap.String("reason", reason),
		)
	})
	h.SetJudgeSubmitter(judgeQueue)

	ctx := context.Background()
	if err = judgeQueue.Consume(ctx, h.ConsumeJudge); err != nil {
		logger.Warn("consume judge queue failed, fallback local async judge", zap.Error(err))
		_ = judgeQueue.Close()
		return
	}

	logger.Info("judge queue consumer started", zap.String("queue", cfg.RabbitMQ.JudgeQueue))
	h.StartPendingSubmitRequeue(ctx)
}
