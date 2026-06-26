package router

import (
	"context"
	"log"

	"kkg-backend/internal/config"
	ojhandler "kkg-backend/internal/oj/handler"
	"kkg-backend/internal/oj/infra/mq"
)

func startOJJudgeQueue(cfg config.OJConfig, h *ojhandler.Handler) {
	if !cfg.RabbitMQ.Enabled {
		return
	}
	judgeQueue, err := mq.NewJudgeQueue(cfg.RabbitMQ.URL, cfg.RabbitMQ.JudgeQueue)
	if err != nil {
		log.Printf("warn: init rabbitmq failed, fallback local async judge: %v", err)
		return
	}
	judgeQueue.SetDeadLetterHandler(func(submitID int64, reason string, retryCount int32) {
		h.MarkSubmitSystemError(submitID, reason)
		log.Printf("error: judge submit moved to dlq submit_id=%d retry_count=%d reason=%s", submitID, retryCount, reason)
	})
	h.SetJudgeSubmitter(judgeQueue)

	ctx := context.Background()
	if err = judgeQueue.Consume(ctx, h.ConsumeJudge); err != nil {
		log.Printf("warn: consume judge queue failed, fallback local async judge: %v", err)
		_ = judgeQueue.Close()
		return
	}

	log.Printf("judge queue consumer started queue=%s", cfg.RabbitMQ.JudgeQueue)
	h.StartPendingSubmitRequeue(ctx)
}
