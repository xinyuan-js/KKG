package handler

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"strings"
	"time"
	"kkg-backend/internal/oj/common"
	"kkg-backend/internal/oj/model/entity"
)

func (h *Handler) QuestionRun(c *gin.Context) {
	_ = h.mustLoginUser(c)
	var req struct {
		Language   string `json:"language"`
		Code       string `json:"code"`
		QuestionID int64  `json:"questionId"`
		Input      string `json:"input"`
	}
	mustBindJSON(c, &req)
	if req.QuestionID <= 0 || strings.TrimSpace(req.Language) == "" || strings.TrimSpace(req.Code) == "" {
		panic(common.NewBizError(common.ParamsError, "请求参数错误"))
	}
	lang := strings.ToLower(strings.TrimSpace(req.Language))
	if lang != "go" {
		panic(common.NewBizError(common.ParamsError, "当前仅支持 Go 语言"))
	}
	var question entity.Question
	mustNoErr(h.db.Where("id=? AND isDelete=0", req.QuestionID).First(&question).Error)
	resp, err := h.executeCode(req.Language, req.Code, []string{req.Input})
	mustNoErr(err)
	output := ""
	if len(resp.OutputList) > 0 {
		output = resp.OutputList[0]
	}
	c.JSON(http.StatusOK, common.Success(map[string]interface{}{
		"output":    output,
		"judgeInfo": resp.JudgeInfo,
	}))
}
func (h *Handler) QuestionSubmitDo(c *gin.Context) {
	login := h.mustLoginUser(c)
	var req struct {
		Language   string `json:"language"`
		Code       string `json:"code"`
		QuestionID int64  `json:"questionId"`
	}
	mustBindJSON(c, &req)
	if req.QuestionID <= 0 || strings.TrimSpace(req.Language) == "" || strings.TrimSpace(req.Code) == "" {
		panic(common.NewBizError(common.ParamsError, "请求参数错误"))
	}
	lang := strings.ToLower(strings.TrimSpace(req.Language))
	if lang != "go" {
		panic(common.NewBizError(common.ParamsError, "当前仅支持 Go 语言"))
	}
	var question entity.Question
	mustNoErr(h.db.Where("id=? AND isDelete=0", req.QuestionID).First(&question).Error)

	// 频率限制：同一用户 5 秒内仅允许提交一次，防止刷提交导致队列拥塞。
	var latest entity.QuestionSubmit
	if err := h.db.Where("userId=? AND isDelete=0", login.ID).Order("id desc").First(&latest).Error; err == nil {
		if time.Since(latest.CreateTime) < 5*time.Second {
			panic(common.NewBizError(common.OperationError, "提交过于频繁，请 5 秒后再试"))
		}
	}

	qs := entity.QuestionSubmit{Language: req.Language, Code: req.Code, QuestionID: req.QuestionID, UserID: login.ID, Status: 0, JudgeInfo: "{}"}
	mustNoErr(h.db.Create(&qs).Error)
	_ = h.db.Model(&entity.Question{}).Where("id=?", req.QuestionID).Update("submitNum", gorm.Expr("submitNum + 1")).Error
	if h.judgeSubmitter != nil {
		if err := h.judgeSubmitter.Publish(qs.ID); err != nil {
			go h.judgeAsync(qs.ID)
		}
	} else {
		go h.judgeAsync(qs.ID)
	}
	c.JSON(http.StatusOK, common.Success(qs.ID))
}
func (h *Handler) QuestionSubmitList(c *gin.Context) {
	login := h.mustLoginUser(c)
	var req struct {
		common.PageRequest
		QuestionID int64 `json:"questionId"`
		UserID     int64 `json:"userId"`
		Status     int32 `json:"status"`
	}
	mustBindJSON(c, &req)
	req.Normalize()
	var list []entity.QuestionSubmit
	query := h.db.Model(&entity.QuestionSubmit{}).Where("isDelete=0")
	if req.QuestionID > 0 {
		query = query.Where("questionId=?", req.QuestionID)
	}
	// 普通用户只能查看自己的提交；管理员/超级管理员可查看所有人提交。
	if !h.userSvc.IsAdmin(login) {
		req.UserID = login.ID
	}
	if req.UserID > 0 {
		query = query.Where("userId=?", req.UserID)
	}
	if req.PageSize > 20 {
		panic(common.NewBizError(common.ParamsError, "请求参数错误"))
	}
	var total int64
	mustNoErr(query.Count(&total).Error)
	mustNoErr(query.Offset(int((req.Current - 1) * req.PageSize)).Limit(int(req.PageSize)).Order("id desc").Find(&list).Error)
	resp := make([]map[string]interface{}, 0, len(list))
	for _, it := range list {
		code := it.Code
		if it.UserID != login.ID && !h.userSvc.IsAdmin(login) {
			code = ""
		}
		status := it.Status
		judgeInfo := it.JudgeInfo
		if status <= 1 {
			if runtime, ok := h.getSubmitRuntimeStatus(it.ID); ok {
				status = runtime.Status
				judgeInfo = runtime.judgeInfoJSON()
			}
		}
		resp = append(resp, map[string]interface{}{"id": it.ID, "language": it.Language, "code": code, "judgeInfo": json.RawMessage(judgeInfo), "status": status, "questionId": it.QuestionID, "userId": it.UserID, "createTime": it.CreateTime, "updateTime": it.UpdateTime})
	}
	c.JSON(http.StatusOK, common.Success(common.PageResult{Records: resp, Total: total, Current: req.Current, Size: req.PageSize}))
}

func (h *Handler) SubmissionEvents(c *gin.Context) {
	login := h.mustLoginUser(c)
	eventCh := h.subscribeSubmitEvents(login.ID)
	defer h.unsubscribeSubmitEvents(login.ID, eventCh)

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.AbortWithStatusJSON(http.StatusInternalServerError, common.Error(common.SystemError, "stream not supported"))
		return
	}
	flusher.Flush()

	keepAlive := time.NewTicker(20 * time.Second)
	defer keepAlive.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-keepAlive.C:
			_, _ = c.Writer.Write([]byte(": ping\n\n"))
			flusher.Flush()
		case evt := <-eventCh:
			data, err := json.Marshal(evt)
			if err != nil {
				continue
			}
			_, _ = c.Writer.Write([]byte("event: submission\n"))
			_, _ = c.Writer.Write([]byte("data: "))
			_, _ = c.Writer.Write(data)
			_, _ = c.Writer.Write([]byte("\n\n"))
			flusher.Flush()
		}
	}
}
