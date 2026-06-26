package handler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"yuoj-go-backend/internal/config"
	"yuoj-go-backend/internal/infra/cache"
	"yuoj-go-backend/internal/model/entity"
	"yuoj-go-backend/internal/service"

	"github.com/gomodule/redigo/redis"
	"gorm.io/gorm"
)

const agentPasswordSalt = "yupi"

type Handler struct {
	db             *gorm.DB
	userSvc        *service.UserService
	cfg            *config.Config
	redisPool      *redis.Pool
	judgeSubmitter JudgeSubmitter
	submitMu       sync.RWMutex
	submitSubs     map[int64]map[chan submitEvent]struct{}
}

const (
	firstACRankZSetKey   = "oj:rank:first_ac:24h"
	firstACExpireZSetKey = "oj:rank:first_ac:24h:events"
	firstACEventTTL      = 24 * time.Hour
	submitRuntimeTTL     = 30 * time.Minute
	submitFinalCacheTTL  = 5 * time.Minute

	submitStatusPending     int32 = 0
	submitStatusRunning     int32 = 1
	submitStatusAccepted    int32 = 2
	submitStatusRejected    int32 = 3
	submitStatusSystemError int32 = 4
)

type JudgeSubmitter interface {
	Publish(submitID int64) error
}

func New(db *gorm.DB, userSvc *service.UserService, cfg *config.Config) *Handler {
	return &Handler{
		db:         db,
		userSvc:    userSvc,
		cfg:        cfg,
		redisPool:  cache.NewRedisPool(cfg.Redis),
		submitSubs: make(map[int64]map[chan submitEvent]struct{}),
	}
}

type submitEvent struct {
	SubmitID   int64  `json:"submitId"`
	QuestionID int64  `json:"questionId"`
	Status     int32  `json:"status"`
	Message    string `json:"message"`
	Score      int64  `json:"score,omitempty"`
	Time       int64  `json:"time,omitempty"`
	Memory     int64  `json:"memory,omitempty"`
	Progress   int64  `json:"progress,omitempty"`
	OccurredAt int64  `json:"occurredAt"`
}

func (h *Handler) SetJudgeSubmitter(s JudgeSubmitter) {
	h.judgeSubmitter = s
}

func (h *Handler) StartPendingSubmitRequeue(ctx context.Context) {
	if h.judgeSubmitter == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				h.requeueStuckSubmits()
			}
		}
	}()
}

func (h *Handler) requeueStuckSubmits() {
	if h.judgeSubmitter == nil {
		return
	}
	now := time.Now()
	var pending []entity.QuestionSubmit
	if err := h.db.
		Where("status = ? AND isDelete = 0 AND createTime < ?", submitStatusPending, now.Add(-30*time.Second)).
		Order("id asc").
		Limit(100).
		Find(&pending).Error; err == nil {
		for _, item := range pending {
			_ = h.judgeSubmitter.Publish(item.ID)
		}
	}

	var running []entity.QuestionSubmit
	if err := h.db.
		Where("status = ? AND isDelete = 0 AND updateTime < ?", submitStatusRunning, now.Add(-10*time.Minute)).
		Order("id asc").
		Limit(100).
		Find(&running).Error; err != nil {
		return
	}
	for _, item := range running {
		reset := h.db.Model(&entity.QuestionSubmit{}).
			Where("id = ? AND status = ? AND isDelete = 0", item.ID, submitStatusRunning).
			Updates(map[string]interface{}{"status": submitStatusPending, "judgeInfo": `{"message":"Requeued after judge timeout"}`, "updateTime": now})
		if reset.Error == nil && reset.RowsAffected == 1 {
			_ = h.judgeSubmitter.Publish(item.ID)
		}
	}
}

func (h *Handler) ConsumeJudge(submitID int64) error {
	if submitID <= 0 {
		return errors.New("invalid submit id")
	}
	return h.judgeAsync(submitID)
}

func (h *Handler) MarkSubmitSystemError(submitID int64, reason string) {
	if submitID <= 0 {
		return
	}
	msg := strings.TrimSpace(reason)
	if msg == "" {
		msg = "System retry exhausted"
	}
	judgeInfo := fmt.Sprintf(`{"message":"%s"}`, escapeJSON(msg))
	res := h.db.Model(&entity.QuestionSubmit{}).
		Where("id = ? AND status IN ? AND isDelete = 0", submitID, []int32{submitStatusPending, submitStatusRunning}).
		Updates(map[string]interface{}{
			"status":     submitStatusSystemError,
			"judgeInfo":  judgeInfo,
			"updateTime": time.Now(),
		})
	if res.Error != nil || res.RowsAffected == 0 {
		return
	}
	var s entity.QuestionSubmit
	if err := h.db.Select("id, questionId, userId").Where("id = ? AND isDelete = 0", submitID).First(&s).Error; err != nil {
		return
	}
	h.publishSubmitRuntimeEvent(s.UserID, submitEvent{
		SubmitID:   s.ID,
		QuestionID: s.QuestionID,
		Status:     submitStatusSystemError,
		Message:    msg,
		Progress:   100,
		OccurredAt: time.Now().UnixMilli(),
	}, submitFinalCacheTTL)
}

func (h *Handler) UserService() *service.UserService {
	return h.userSvc
}

func (h *Handler) JWTSecret() string {
	return strings.TrimSpace(h.cfg.JWTSecret)
}
