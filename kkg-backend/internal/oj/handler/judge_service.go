package handler

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	judgeintegration "kkg-backend/internal/oj/integration/judge"
	"kkg-backend/internal/oj/model/entity"

	"github.com/gomodule/redigo/redis"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (h *Handler) judgeAsync(submitID int64) error {
	claim := h.db.Model(&entity.QuestionSubmit{}).
		Where("id=? AND status=? AND isDelete=0", submitID, submitStatusPending).
		Updates(map[string]interface{}{"status": submitStatusRunning, "updateTime": time.Now()})
	if claim.Error != nil {
		return claim.Error
	}
	if claim.RowsAffected == 0 {
		return nil
	}
	var s entity.QuestionSubmit
	if err := h.db.Where("id=? AND isDelete=0", submitID).First(&s).Error; err != nil {
		return err
	}
	h.publishSubmitRuntimeEvent(s.UserID, submitEvent{
		SubmitID:   s.ID,
		QuestionID: s.QuestionID,
		Status:     submitStatusRunning,
		Message:    "Judging",
		Progress:   10,
		OccurredAt: time.Now().UnixMilli(),
	}, submitRuntimeTTL)
	var q entity.Question
	if h.db.Where("id=? AND isDelete=0", s.QuestionID).First(&q).Error != nil {
		judgeInfo := `{"message":"Question Not Found"}`
		if err := h.finishJudgeSubmit(submitID, submitStatusRejected, judgeInfo); err != nil {
			return err
		}
		h.publishSubmitRuntimeEvent(s.UserID, submitEvent{
			SubmitID:   s.ID,
			QuestionID: s.QuestionID,
			Status:     submitStatusRejected,
			Message:    "Question Not Found",
			Progress:   100,
			OccurredAt: time.Now().UnixMilli(),
		}, submitFinalCacheTTL)
		return nil
	}
	var judgeCases []judgeCase
	_ = json.Unmarshal([]byte(q.JudgeCase), &judgeCases)
	inputs := make([]string, 0, len(judgeCases))
	for _, jc := range judgeCases {
		inputs = append(inputs, jc.Input)
	}
	resp, err := h.executeCode(s.Language, s.Code, inputs)
	if err != nil {
		judgeInfo := fmt.Sprintf(`{"message":"%s"}`, escapeJSON(err.Error()))
		if err := h.finishJudgeSubmit(submitID, submitStatusSystemError, judgeInfo); err != nil {
			return err
		}
		h.publishSubmitRuntimeEvent(s.UserID, submitEvent{
			SubmitID:   s.ID,
			QuestionID: s.QuestionID,
			Status:     submitStatusSystemError,
			Message:    err.Error(),
			Progress:   100,
			OccurredAt: time.Now().UnixMilli(),
		}, submitFinalCacheTTL)
		return nil
	}
	result := doJudge(judgeCases, resp.OutputList, resp.JudgeInfo, q.JudgeConfig)
	status := int32(submitStatusRejected)
	if strings.EqualFold(result.Message, "Accepted") {
		status = submitStatusAccepted
	}
	jInfo, _ := json.Marshal(result)
	shouldRecordFirstAC := false
	if status == submitStatusAccepted {
		var err error
		shouldRecordFirstAC, err = h.finishAcceptedJudgeSubmit(s, string(jInfo))
		if err != nil {
			return err
		}
	} else if err := h.finishJudgeSubmit(submitID, status, string(jInfo)); err != nil {
		return err
	}
	if shouldRecordFirstAC {
		h.recordFirstAccepted24h(s.UserID, s.QuestionID, s.ID)
	}
	h.publishSubmitRuntimeEvent(s.UserID, submitEvent{
		SubmitID:   s.ID,
		QuestionID: s.QuestionID,
		Status:     status,
		Message:    result.Message,
		Score:      int64(result.Score),
		Time:       int64(result.Time),
		Memory:     int64(result.Memory),
		Progress:   100,
		OccurredAt: time.Now().UnixMilli(),
	}, submitFinalCacheTTL)
	return nil
}

func (h *Handler) finishAcceptedJudgeSubmit(s entity.QuestionSubmit, judgeInfo string) (bool, error) {
	shouldRecordFirstAC := false
	err := h.db.Transaction(func(tx *gorm.DB) error {
		var q entity.Question
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND isDelete = 0", s.QuestionID).
			First(&q).Error; err != nil {
			return err
		}
		var acceptedBefore int64
		if err := tx.Model(&entity.QuestionSubmit{}).
			Where("userId = ? AND questionId = ? AND status = ? AND isDelete = 0", s.UserID, s.QuestionID, submitStatusAccepted).
			Count(&acceptedBefore).Error; err != nil {
			return err
		}
		res := tx.Model(&entity.QuestionSubmit{}).
			Where("id = ? AND status = ? AND isDelete = 0", s.ID, submitStatusRunning).
			Updates(map[string]interface{}{"status": submitStatusAccepted, "judgeInfo": judgeInfo, "updateTime": time.Now()})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		if err := tx.Model(&entity.Question{}).
			Where("id = ?", s.QuestionID).
			Update("acceptedNum", gorm.Expr("acceptedNum + 1")).Error; err != nil {
			return err
		}
		shouldRecordFirstAC = acceptedBefore == 0
		return nil
	})
	return shouldRecordFirstAC, err
}

func (h *Handler) finishJudgeSubmit(submitID int64, status int32, judgeInfo string) error {
	res := h.db.Model(&entity.QuestionSubmit{}).
		Where("id=? AND status=? AND isDelete=0", submitID, submitStatusRunning).
		Updates(map[string]interface{}{"status": status, "judgeInfo": judgeInfo, "updateTime": time.Now()})
	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (h *Handler) subscribeSubmitEvents(userID int64) chan submitEvent {
	ch := make(chan submitEvent, 16)
	h.submitMu.Lock()
	defer h.submitMu.Unlock()
	if _, ok := h.submitSubs[userID]; !ok {
		h.submitSubs[userID] = make(map[chan submitEvent]struct{})
	}
	h.submitSubs[userID][ch] = struct{}{}
	return ch
}

func (h *Handler) unsubscribeSubmitEvents(userID int64, ch chan submitEvent) {
	h.submitMu.Lock()
	defer h.submitMu.Unlock()
	subs, ok := h.submitSubs[userID]
	if !ok {
		return
	}
	delete(subs, ch)
	if len(subs) == 0 {
		delete(h.submitSubs, userID)
	}
	close(ch)
}

func (h *Handler) publishSubmitEvent(userID int64, evt submitEvent) {
	h.submitMu.RLock()
	subs, ok := h.submitSubs[userID]
	if !ok || len(subs) == 0 {
		h.submitMu.RUnlock()
		return
	}
	targets := make([]chan submitEvent, 0, len(subs))
	for ch := range subs {
		targets = append(targets, ch)
	}
	h.submitMu.RUnlock()
	for _, ch := range targets {
		select {
		case ch <- evt:
		default:
		}
	}
}

func (h *Handler) publishSubmitRuntimeEvent(userID int64, evt submitEvent, ttl time.Duration) {
	h.setSubmitRuntimeStatus(userID, evt, ttl)
	h.publishSubmitEvent(userID, evt)
}

func submitRuntimeKey(submitID int64) string {
	return fmt.Sprintf("oj:submit:runtime:%d", submitID)
}

type submitRuntimeStatus struct {
	SubmitID   int64  `json:"submitId"`
	QuestionID int64  `json:"questionId"`
	UserID     int64  `json:"userId"`
	Status     int32  `json:"status"`
	Message    string `json:"message"`
	Score      int64  `json:"score,omitempty"`
	Time       int64  `json:"time,omitempty"`
	Memory     int64  `json:"memory,omitempty"`
	Progress   int64  `json:"progress,omitempty"`
	OccurredAt int64  `json:"occurredAt"`
}

func (s submitRuntimeStatus) judgeInfoJSON() string {
	payload := map[string]interface{}{
		"message":  s.Message,
		"score":    s.Score,
		"time":     s.Time,
		"memory":   s.Memory,
		"progress": s.Progress,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func (h *Handler) setSubmitRuntimeStatus(userID int64, evt submitEvent, ttl time.Duration) {
	if h.redisPool == nil || evt.SubmitID <= 0 || ttl <= 0 {
		return
	}
	conn := h.redisPool.Get()
	defer conn.Close()
	status := submitRuntimeStatus{
		SubmitID:   evt.SubmitID,
		QuestionID: evt.QuestionID,
		UserID:     userID,
		Status:     evt.Status,
		Message:    evt.Message,
		Score:      evt.Score,
		Time:       evt.Time,
		Memory:     evt.Memory,
		Progress:   evt.Progress,
		OccurredAt: evt.OccurredAt,
	}
	raw, err := json.Marshal(status)
	if err != nil {
		return
	}
	_, _ = conn.Do("SETEX", submitRuntimeKey(evt.SubmitID), int(ttl.Seconds()), raw)
}

func (h *Handler) getSubmitRuntimeStatus(submitID int64) (submitRuntimeStatus, bool) {
	if h.redisPool == nil || submitID <= 0 {
		return submitRuntimeStatus{}, false
	}
	conn := h.redisPool.Get()
	defer conn.Close()
	raw, err := redis.Bytes(conn.Do("GET", submitRuntimeKey(submitID)))
	if err != nil {
		return submitRuntimeStatus{}, false
	}
	var status submitRuntimeStatus
	if err := json.Unmarshal(raw, &status); err != nil {
		return submitRuntimeStatus{}, false
	}
	if status.SubmitID != submitID {
		return submitRuntimeStatus{}, false
	}
	return status, true
}

type judgeCase = judgeintegration.Case

func (h *Handler) executeCode(language, code string, inputList []string) (*judgeintegration.ExecuteCodeResp, error) {
	return judgeintegration.ExecuteCode(judgeintegration.Config{
		SandboxType: h.cfg.Judge.SandboxType,
		SandboxURL:  h.cfg.Judge.SandboxURL,
		AuthSecret:  h.cfg.Judge.AuthSecret,
	}, language, code, inputList)
}

func doJudge(cases []judgeCase, outputs []string, sandbox judgeintegration.Info, judgeConfigJSON string) judgeintegration.Info {
	return judgeintegration.Judge(cases, outputs, sandbox, judgeConfigJSON)
}

func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}
