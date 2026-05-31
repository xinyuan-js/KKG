package handler

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"yuoj-go-backend/internal/common"
	"yuoj-go-backend/internal/config"
	"yuoj-go-backend/internal/model/entity"
	"yuoj-go-backend/internal/service"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
	"github.com/tencentyun/cos-go-sdk-v5"
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
	pool := &redis.Pool{
		MaxIdle:     5,
		IdleTimeout: 5 * time.Minute,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", cfg.Redis.Addr)
			if err != nil {
				return nil, err
			}
			if strings.TrimSpace(cfg.Redis.Password) != "" {
				if _, err = c.Do("AUTH", cfg.Redis.Password); err != nil {
					msg := strings.ToLower(err.Error())
					if !strings.Contains(msg, "without any password configured") {
						_ = c.Close()
						return nil, err
					}
				}
			}
			if cfg.Redis.DB > 0 {
				if _, err = c.Do("SELECT", cfg.Redis.DB); err != nil {
					_ = c.Close()
					return nil, err
				}
			}
			return c, nil
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < 30*time.Second {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}
	return &Handler{
		db:         db,
		userSvc:    userSvc,
		cfg:        cfg,
		redisPool:  pool,
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

type userRegisterReq struct{ UserAccount, UserPassword, CheckPassword string }
type userLoginReq struct{ UserAccount, UserPassword string }

func (h *Handler) UserRegister(c *gin.Context) {
	var req userRegisterReq
	mustBindJSON(c, &req)
	id, err := h.userSvc.Register(req.UserAccount, req.UserPassword, req.CheckPassword)
	mustNoErr(err)
	c.JSON(http.StatusOK, common.Success(id))
}

func (h *Handler) UserLogin(c *gin.Context) {
	var req userLoginReq
	mustBindJSON(c, &req)
	u, err := h.userSvc.Login(req.UserAccount, req.UserPassword)
	mustNoErr(err)
	sess := sessions.Default(c)
	sess.Set(common.UserLoginState, u.ID)
	_ = sess.Save()
	c.JSON(http.StatusOK, common.Success(loginUserVO(u)))
}

func (h *Handler) UserLogout(c *gin.Context) {
	sess := sessions.Default(c)
	sess.Delete(common.UserLoginState)
	_ = sess.Save()
	c.JSON(http.StatusOK, common.Success(true))
}

func (h *Handler) UserGetLogin(c *gin.Context) {
	c.JSON(http.StatusOK, common.Success(loginUserVO(h.mustLoginUser(c))))
}
func (h *Handler) UserWxLogin(c *gin.Context) {
	code := strings.TrimSpace(c.Query("code"))
	if code == "" {
		panic(common.NewBizError(common.ParamsError, "请求参数错误"))
	}
	unionID := "wx_union_" + code
	mpOpenID := "wx_open_" + code
	if h.cfg.WX.OpenAppID != "" && h.cfg.WX.OpenSecret != "" {
		realUnion, realOpen, err := h.fetchWxOpenUserInfo(code)
		if err == nil {
			if realUnion != "" {
				unionID = realUnion
			}
			if realOpen != "" {
				mpOpenID = realOpen
			}
		}
	}
	var u entity.User
	err := h.db.Where("unionId = ? AND isDelete = 0", unionID).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		u = entity.User{
			UnionID:    unionID,
			MpOpenID:   mpOpenID,
			UserName:   "wx_" + code,
			UserRole:   common.DefaultRole,
			UserAvatar: "",
		}
		mustNoErr(h.db.Create(&u).Error)
	} else {
		mustNoErr(err)
	}
	if u.UserRole == common.BanRole {
		panic(common.NewBizError(common.ForbiddenError, "该用户已被封，禁止登录"))
	}
	sess := sessions.Default(c)
	sess.Set(common.UserLoginState, u.ID)
	_ = sess.Save()
	c.JSON(http.StatusOK, common.Success(loginUserVO(&u)))
}

func (h *Handler) UserAdd(c *gin.Context) {
	h.mustAdmin(c)
	var u entity.User
	mustBindJSON(c, &u)
	mustNoErr(h.userSvc.CreateByAdmin(&u))
	c.JSON(http.StatusOK, common.Success(u.ID))
}

func (h *Handler) UserDelete(c *gin.Context) {
	login := h.mustLoginUser(c)
	var req common.DeleteRequest
	mustBindJSON(c, &req)
	mustNoErr(h.userSvc.SoftDeleteBySuperAdmin(login, req.ID))
	c.JSON(http.StatusOK, common.Success(true))
}

func (h *Handler) UserUpdate(c *gin.Context) {
	login := h.mustLoginUser(c)
	var u entity.User
	mustBindJSON(c, &u)
	mustNoErr(h.userSvc.UpdateByAdmin(login, &u))
	c.JSON(http.StatusOK, common.Success(true))
}

func (h *Handler) UserGet(c *gin.Context) {
	h.mustAdmin(c)
	id := parseIDQuery(c)
	u, err := h.userSvc.GetByID(id)
	mustNoErr(err)
	c.JSON(http.StatusOK, common.Success(u))
}
func (h *Handler) UserGetVO(c *gin.Context) {
	id := parseIDQuery(c)
	u, err := h.userSvc.GetByID(id)
	mustNoErr(err)
	c.JSON(http.StatusOK, common.Success(userVO(u)))
}
func (h *Handler) UserList(c *gin.Context) {
	h.mustAdmin(c)
	var req common.PageRequest
	mustBindJSON(c, &req)
	req.Normalize()
	list, total, err := h.userSvc.List(req.Current, req.PageSize)
	mustNoErr(err)
	c.JSON(http.StatusOK, common.Success(common.PageResult{Records: list, Total: total, Current: req.Current, Size: req.PageSize}))
}
func (h *Handler) UserListVO(c *gin.Context) {
	var req common.PageRequest
	mustBindJSON(c, &req)
	req.Normalize()
	if req.PageSize > 20 {
		panic(common.NewBizError(common.ParamsError, "请求参数错误"))
	}
	list, total, err := h.userSvc.List(req.Current, req.PageSize)
	mustNoErr(err)
	vos := make([]map[string]interface{}, 0, len(list))
	for i := range list {
		vos = append(vos, userVO(&list[i]))
	}
	c.JSON(http.StatusOK, common.Success(common.PageResult{Records: vos, Total: total, Current: req.Current, Size: req.PageSize}))
}
func (h *Handler) UserUpdateMy(c *gin.Context) {
	login := h.mustLoginUser(c)
	var req entity.User
	mustBindJSON(c, &req)
	req.ID = login.ID
	mustNoErr(h.userSvc.UpdateByID(&req))
	c.JSON(http.StatusOK, common.Success(true))
}

func (h *Handler) QuestionAdd(c *gin.Context) {
	login := h.mustLoginUser(c)
	var q entity.Question
	mustBindJSON(c, &q)
	validQuestionInput(&q, true)
	q.UserID = login.ID
	q.ThumbNum, q.FavourNum, q.SubmitNum, q.AcceptedNum = 0, 0, 0, 0
	mustNoErr(h.db.Create(&q).Error)
	c.JSON(http.StatusOK, common.Success(q.ID))
}
func (h *Handler) QuestionDelete(c *gin.Context) {
	login := h.mustLoginUser(c)
	var req common.DeleteRequest
	mustBindJSON(c, &req)
	var q entity.Question
	mustNoErr(h.db.Where("id=? AND isDelete=0", req.ID).First(&q).Error)
	if q.UserID != login.ID && !h.userSvc.IsAdmin(login) {
		panic(common.NewBizError(common.NoAuthError, "无权限"))
	}
	mustNoErr(h.db.Model(&entity.Question{}).Where("id=?", req.ID).Update("isDelete", 1).Error)
	c.JSON(http.StatusOK, common.Success(true))
}
func (h *Handler) QuestionUpdate(c *gin.Context) {
	h.mustAdmin(c)
	var q entity.Question
	mustBindJSON(c, &q)
	validQuestionInput(&q, false)
	mustNoErr(h.db.Model(&entity.Question{}).Where("id=?", q.ID).Updates(&q).Error)
	c.JSON(http.StatusOK, common.Success(true))
}
func (h *Handler) QuestionGet(c *gin.Context) {
	login := h.mustLoginUser(c)
	id := parseIDQuery(c)
	var q entity.Question
	mustNoErr(h.db.Where("id=? AND isDelete=0", id).First(&q).Error)
	if q.UserID != login.ID && !h.userSvc.IsAdmin(login) {
		panic(common.NewBizError(common.NoAuthError, "无权限"))
	}
	c.JSON(http.StatusOK, common.Success(q))
}
func (h *Handler) QuestionGetVO(c *gin.Context) {
	id := parseIDQuery(c)
	var q entity.Question
	mustNoErr(h.db.Where("id=? AND isDelete=0", id).First(&q).Error)
	c.JSON(http.StatusOK, common.Success(questionVO(&q)))
}
func (h *Handler) QuestionListVO(c *gin.Context) { h.questionList(c, false, false) }
func (h *Handler) QuestionMyListVO(c *gin.Context) {
	login := h.mustLoginUser(c)
	h.questionList(c, true, false, login.ID)
}
func (h *Handler) QuestionList(c *gin.Context) { h.mustAdmin(c); h.questionList(c, false, true) }
func (h *Handler) QuestionEdit(c *gin.Context) {
	login := h.mustLoginUser(c)
	var q entity.Question
	mustBindJSON(c, &q)
	validQuestionInput(&q, false)
	var old entity.Question
	mustNoErr(h.db.Where("id=? AND isDelete=0", q.ID).First(&old).Error)
	if old.UserID != login.ID && !h.userSvc.IsAdmin(login) {
		panic(common.NewBizError(common.NoAuthError, "无权限"))
	}
	mustNoErr(h.db.Model(&entity.Question{}).Where("id=?", q.ID).Updates(&q).Error)
	c.JSON(http.StatusOK, common.Success(true))
}

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

func (h *Handler) QuestionFirstACRank24h(c *gin.Context) {
	limit := int64(20)
	if v := strings.TrimSpace(c.Query("limit")); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	conn := h.redisPool.Get()
	defer conn.Close()
	if err := h.applyExpiredFirstACEvents(conn); err != nil {
		c.JSON(http.StatusOK, common.Success(map[string]interface{}{
			"windowHours": 24,
			"records":     []interface{}{},
		}))
		return
	}
	values, err := redis.Values(conn.Do("ZREVRANGE", firstACRankZSetKey, 0, limit-1, "WITHSCORES"))
	if err != nil {
		c.JSON(http.StatusOK, common.Success(map[string]interface{}{
			"windowHours": 24,
			"records":     []interface{}{},
		}))
		return
	}
	type rankItem struct {
		UserID       int64  `json:"userId"`
		BlogUserID   int64  `json:"blogUserId"`
		UserName     string `json:"userName"`
		UserAvatar   string `json:"userAvatar"`
		FirstACCount int64  `json:"firstAcCount"`
		Rank         int64  `json:"rank"`
	}
	items := make([]rankItem, 0, len(values)/2)
	for i := 0; i+1 < len(values); i += 2 {
		member, _ := redis.String(values[i], nil)
		score, _ := redis.Int64(values[i+1], nil)
		uid, err := strconv.ParseInt(member, 10, 64)
		if err != nil {
			continue
		}
		var u entity.User
		if err := h.db.Where("id = ? AND isDelete = 0", uid).First(&u).Error; err != nil {
			continue
		}
		var blogUserID int64
		_ = h.db.Table("users").Where("username = ? AND status <> -1", strings.TrimSpace(u.UserAccount)).Select("id").Scan(&blogUserID).Error
		items = append(items, rankItem{
			UserID:       uid,
			BlogUserID:   blogUserID,
			UserName:     u.UserName,
			UserAvatar:   u.UserAvatar,
			FirstACCount: score,
		})
	}
	for i := range items {
		items[i].Rank = int64(i + 1)
	}
	c.JSON(http.StatusOK, common.Success(map[string]interface{}{
		"windowHours": 24,
		"records":     items,
	}))
}

func (h *Handler) recordFirstAccepted24h(userID, questionID, submitID int64) {
	conn := h.redisPool.Get()
	defer conn.Close()
	if err := h.applyExpiredFirstACEvents(conn); err != nil {
		return
	}
	member := fmt.Sprintf("%d:%d:%d", userID, questionID, submitID)
	expireAt := time.Now().Add(firstACEventTTL).Unix()
	if _, err := conn.Do("MULTI"); err != nil {
		return
	}
	_ = conn.Send("ZINCRBY", firstACRankZSetKey, 1, strconv.FormatInt(userID, 10))
	_ = conn.Send("ZADD", firstACExpireZSetKey, expireAt, member)
	_ = conn.Send("EXPIRE", firstACRankZSetKey, int(firstACEventTTL.Seconds())+3600)
	_ = conn.Send("EXPIRE", firstACExpireZSetKey, int(firstACEventTTL.Seconds())+3600)
	_, _ = conn.Do("EXEC")
}

func (h *Handler) applyExpiredFirstACEvents(conn redis.Conn) error {
	now := time.Now().Unix()
	for {
		values, err := redis.Values(conn.Do("ZRANGEBYSCORE", firstACExpireZSetKey, "-inf", now, "LIMIT", 0, 200))
		if err != nil {
			return err
		}
		if len(values) == 0 {
			return nil
		}
		for _, raw := range values {
			member, err := redis.String(raw, nil)
			if err != nil {
				continue
			}
			parts := strings.Split(member, ":")
			if len(parts) < 1 {
				continue
			}
			userID := parts[0]
			_, _ = conn.Do("ZINCRBY", firstACRankZSetKey, -1, userID)
			_, _ = conn.Do("ZREM", firstACExpireZSetKey, member)
		}
		_, _ = conn.Do("ZREMRANGEBYSCORE", firstACRankZSetKey, "-inf", 0)
	}
}

func (h *Handler) QuestionSolutionBind(c *gin.Context) {
	login := h.mustLoginUser(c)
	var req struct {
		QuestionID int64 `json:"questionId"`
		PostID     int64 `json:"postId"`
	}
	mustBindJSON(c, &req)
	if req.QuestionID <= 0 || req.PostID <= 0 {
		panic(common.NewBizError(common.ParamsError, "请求参数错误"))
	}
	var q entity.Question
	mustNoErr(h.db.Where("id=? AND isDelete=0", req.QuestionID).First(&q).Error)
	var old entity.QuestionSolutionPost
	err := h.db.Where("questionId=? AND postId=? AND isDelete=0", req.QuestionID, req.PostID).First(&old).Error
	if err == nil {
		c.JSON(http.StatusOK, common.Success(old.ID))
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		mustNoErr(err)
	}
	item := entity.QuestionSolutionPost{
		QuestionID: req.QuestionID,
		PostID:     req.PostID,
		UserID:     login.ID,
		IsDelete:   0,
	}
	mustNoErr(h.db.Create(&item).Error)
	c.JSON(http.StatusOK, common.Success(item.ID))
}

func (h *Handler) QuestionSolutionUnbind(c *gin.Context) {
	login := h.mustLoginUser(c)
	var req struct {
		QuestionID int64 `json:"questionId"`
		PostID     int64 `json:"postId"`
	}
	mustBindJSON(c, &req)
	if req.QuestionID <= 0 || req.PostID <= 0 {
		panic(common.NewBizError(common.ParamsError, "请求参数错误"))
	}
	var q entity.Question
	mustNoErr(h.db.Where("id=? AND isDelete=0", req.QuestionID).First(&q).Error)
	var rel entity.QuestionSolutionPost
	mustNoErr(h.db.Where("questionId=? AND postId=? AND isDelete=0", req.QuestionID, req.PostID).First(&rel).Error)
	if rel.UserID != login.ID && q.UserID != login.ID && !h.userSvc.IsAdmin(login) {
		panic(common.NewBizError(common.NoAuthError, "无权限"))
	}
	mustNoErr(h.db.Model(&entity.QuestionSolutionPost{}).
		Where("questionId=? AND postId=? AND isDelete=0", req.QuestionID, req.PostID).
		Update("isDelete", 1).Error)
	c.JSON(http.StatusOK, common.Success(true))
}

func (h *Handler) QuestionSolutionList(c *gin.Context) {
	var req struct {
		common.PageRequest
		QuestionID int64 `json:"questionId"`
	}
	mustBindJSON(c, &req)
	req.Normalize()
	if req.QuestionID <= 0 {
		panic(common.NewBizError(common.ParamsError, "请求参数错误"))
	}
	if req.PageSize > 50 {
		panic(common.NewBizError(common.ParamsError, "请求参数错误"))
	}
	var total int64
	query := h.db.Model(&entity.QuestionSolutionPost{}).Where("questionId=? AND isDelete=0", req.QuestionID)
	mustNoErr(query.Count(&total).Error)
	var rows []entity.QuestionSolutionPost
	mustNoErr(query.Offset(int((req.Current - 1) * req.PageSize)).Limit(int(req.PageSize)).Order("id desc").Find(&rows).Error)
	postIDs := make([]int64, 0, len(rows))
	for _, it := range rows {
		postIDs = append(postIDs, it.PostID)
	}
	previewMap := h.fetchBlogPostPreviewMap(postIDs)
	records := make([]map[string]interface{}, 0, len(rows))
	for _, it := range rows {
		records = append(records, map[string]interface{}{
			"id":         it.ID,
			"questionId": it.QuestionID,
			"postId":     it.PostID,
			"userId":     it.UserID,
			"createTime": it.CreateTime,
			"post":       previewMap[it.PostID],
		})
	}
	c.JSON(http.StatusOK, common.Success(common.PageResult{
		Records: records,
		Total:   total,
		Current: req.Current,
		Size:    req.PageSize,
	}))
}

func (h *Handler) fetchBlogPostPreviewMap(postIDs []int64) map[int64]map[string]interface{} {
	result := make(map[int64]map[string]interface{}, len(postIDs))
	if len(postIDs) == 0 {
		return result
	}
	baseURL := strings.TrimSpace(h.cfg.Blog.BaseURL)
	if baseURL == "" {
		return result
	}
	baseURL = strings.TrimRight(baseURL, "/")
	client := &http.Client{Timeout: 2 * time.Second}
	for _, postID := range postIDs {
		if postID <= 0 {
			continue
		}
		url := fmt.Sprintf("%s/api/v1/posts/%d", baseURL, postID)
		resp, err := client.Get(url)
		if err != nil || resp == nil {
			continue
		}
		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil || resp.StatusCode >= 400 {
			continue
		}
		var envelope struct {
			Code int                    `json:"code"`
			Data map[string]interface{} `json:"data"`
		}
		if json.Unmarshal(body, &envelope) != nil || envelope.Code != 0 || envelope.Data == nil {
			continue
		}
		result[postID] = map[string]interface{}{
			"id":                envelope.Data["id"],
			"title":             envelope.Data["title"],
			"summary":           envelope.Data["summary"],
			"author_id":         envelope.Data["author_id"],
			"author_name":       envelope.Data["author_name"],
			"authorName":        envelope.Data["author_name"],
			"author_avatar_url": envelope.Data["author_avatar_url"],
			"updated_at":        envelope.Data["updated_at"],
		}
	}
	return result
}

func (h *Handler) PostAdd(c *gin.Context) {
	login := h.mustLoginUser(c)
	var p entity.Post
	mustBindJSON(c, &p)
	validPostInput(&p, true)
	p.UserID = login.ID
	mustNoErr(h.db.Create(&p).Error)
	c.JSON(http.StatusOK, common.Success(p.ID))
}
func (h *Handler) PostDelete(c *gin.Context) {
	login := h.mustLoginUser(c)
	var req common.DeleteRequest
	mustBindJSON(c, &req)
	var p entity.Post
	mustNoErr(h.db.Where("id=? AND isDelete=0", req.ID).First(&p).Error)
	if p.UserID != login.ID && !h.userSvc.IsAdmin(login) {
		panic(common.NewBizError(common.NoAuthError, "无权限"))
	}
	mustNoErr(h.db.Model(&entity.Post{}).Where("id=?", req.ID).Update("isDelete", 1).Error)
	c.JSON(http.StatusOK, common.Success(true))
}
func (h *Handler) PostUpdate(c *gin.Context) {
	h.mustAdmin(c)
	var p entity.Post
	mustBindJSON(c, &p)
	validPostInput(&p, false)
	mustNoErr(h.db.Model(&entity.Post{}).Where("id=?", p.ID).Updates(&p).Error)
	c.JSON(http.StatusOK, common.Success(true))
}
func (h *Handler) PostGetVO(c *gin.Context) {
	id := parseIDQuery(c)
	var p entity.Post
	mustNoErr(h.db.Where("id=? AND isDelete=0", id).First(&p).Error)
	c.JSON(http.StatusOK, common.Success(h.postVO(c, &p)))
}
func (h *Handler) PostListVO(c *gin.Context)   { h.postList(c, false) }
func (h *Handler) PostMyListVO(c *gin.Context) { h.postList(c, true) }
func (h *Handler) PostSearchVO(c *gin.Context) { h.postList(c, false) }
func (h *Handler) PostEdit(c *gin.Context) {
	login := h.mustLoginUser(c)
	var p entity.Post
	mustBindJSON(c, &p)
	validPostInput(&p, false)
	var old entity.Post
	mustNoErr(h.db.Where("id=? AND isDelete=0", p.ID).First(&old).Error)
	if old.UserID != login.ID && !h.userSvc.IsAdmin(login) {
		panic(common.NewBizError(common.NoAuthError, "无权限"))
	}
	mustNoErr(h.db.Model(&entity.Post{}).Where("id=?", p.ID).Updates(&p).Error)
	c.JSON(http.StatusOK, common.Success(true))
}
func (h *Handler) PostThumb(c *gin.Context) {
	login := h.mustLoginUser(c)
	var req struct {
		PostID int64 `json:"postId"`
	}
	mustBindJSON(c, &req)
	var post entity.Post
	mustNoErr(h.db.Where("id=? AND isDelete=0", req.PostID).First(&post).Error)
	result := 0
	mustNoErr(h.db.Transaction(func(tx *gorm.DB) error {
		var old entity.PostThumb
		err := tx.Where("postId=? AND userId=?", req.PostID, login.ID).First(&old).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err = tx.Create(&entity.PostThumb{PostID: req.PostID, UserID: login.ID}).Error; err != nil {
				return err
			}
			if err = tx.Model(&entity.Post{}).Where("id=?", req.PostID).Update("thumbNum", gorm.Expr("thumbNum + 1")).Error; err != nil {
				return err
			}
			result = 1
			return nil
		}
		if err != nil {
			return err
		}
		if err = tx.Delete(&old).Error; err != nil {
			return err
		}
		if err = tx.Model(&entity.Post{}).Where("id=? AND thumbNum>0", req.PostID).Update("thumbNum", gorm.Expr("thumbNum - 1")).Error; err != nil {
			return err
		}
		result = -1
		return nil
	}))
	c.JSON(http.StatusOK, common.Success(result))
}
func (h *Handler) PostFavour(c *gin.Context) {
	login := h.mustLoginUser(c)
	var req struct {
		PostID int64 `json:"postId"`
	}
	mustBindJSON(c, &req)
	var post entity.Post
	mustNoErr(h.db.Where("id=? AND isDelete=0", req.PostID).First(&post).Error)
	result := 0
	mustNoErr(h.db.Transaction(func(tx *gorm.DB) error {
		var old entity.PostFavour
		err := tx.Where("postId=? AND userId=?", req.PostID, login.ID).First(&old).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err = tx.Create(&entity.PostFavour{PostID: req.PostID, UserID: login.ID}).Error; err != nil {
				return err
			}
			if err = tx.Model(&entity.Post{}).Where("id=?", req.PostID).Update("favourNum", gorm.Expr("favourNum + 1")).Error; err != nil {
				return err
			}
			result = 1
			return nil
		}
		if err != nil {
			return err
		}
		if err = tx.Delete(&old).Error; err != nil {
			return err
		}
		if err = tx.Model(&entity.Post{}).Where("id=? AND favourNum>0", req.PostID).Update("favourNum", gorm.Expr("favourNum - 1")).Error; err != nil {
			return err
		}
		result = -1
		return nil
	}))
	c.JSON(http.StatusOK, common.Success(result))
}
func (h *Handler) MyFavourPostList(c *gin.Context) {
	login := h.mustLoginUser(c)
	h.listFavourByUserID(c, login.ID)
}
func (h *Handler) FavourPostList(c *gin.Context) {
	var req struct {
		common.PageRequest
		UserID int64 `json:"userId"`
	}
	mustBindJSON(c, &req)
	if req.UserID <= 0 {
		panic(common.NewBizError(common.ParamsError, "请求参数错误"))
	}
	h.listFavourByUserIDWithReq(c, req.UserID, req.PageRequest)
}

func (h *Handler) FileUpload(c *gin.Context) {
	login := h.mustLoginUser(c)
	biz := strings.TrimSpace(c.PostForm("biz"))
	if biz == "" {
		panic(common.NewBizError(common.ParamsError, "请求参数错误"))
	}
	file, err := c.FormFile("file")
	mustNoErr(err)
	validUploadFile(file.Size, file.Filename, biz)
	now := time.Now().Format("20060102150405")
	dst := fmt.Sprintf("/tmp/%s/%d/%s-%s", biz, login.ID, now, filepath.Base(file.Filename))
	if h.cfg.COS.Enabled && h.cfg.COS.BucketURL != "" && h.cfg.COS.SecretID != "" && h.cfg.COS.SecretKey != "" {
		key := fmt.Sprintf("/%s/%d/%s-%s", biz, login.ID, now, filepath.Base(file.Filename))
		url, err := h.uploadToCOS(c, file, key)
		mustNoErr(err)
		c.JSON(http.StatusOK, common.Success(url))
		return
	}
	mustNoErr(os.MkdirAll(filepath.Dir(dst), 0o755))
	mustNoErr(c.SaveUploadedFile(file, dst))
	c.JSON(http.StatusOK, common.Success("file://"+dst))
}
func (h *Handler) WxGet(c *gin.Context) {
	echostr := c.Query("echostr")
	if h.cfg.WX.MpToken == "" {
		c.String(http.StatusOK, echostr)
		return
	}
	signature := c.Query("signature")
	timestamp := c.Query("timestamp")
	nonce := c.Query("nonce")
	if wxCheckSignature(h.cfg.WX.MpToken, signature, timestamp, nonce) {
		c.String(http.StatusOK, echostr)
		return
	}
	c.String(http.StatusOK, "")
}
func (h *Handler) WxPost(c *gin.Context) {
	var msg wxInMessage
	body, _ := io.ReadAll(c.Request.Body)
	if xml.Unmarshal(body, &msg) != nil {
		c.String(http.StatusOK, "")
		return
	}
	reply := "感谢关注"
	if msg.MsgType == "text" && strings.TrimSpace(msg.Content) != "" {
		reply = "收到：" + msg.Content
	}
	if msg.MsgType == "event" && strings.EqualFold(msg.Event, "CLICK") {
		reply = "你点击了菜单：" + msg.EventKey
	}
	out := wxOutMessage{
		XMLName:      xml.Name{Local: "xml"},
		ToUserName:   cdata(msg.FromUserName),
		FromUserName: cdata(msg.ToUserName),
		CreateTime:   time.Now().Unix(),
		MsgType:      cdata("text"),
		Content:      cdata(reply),
	}
	xmlBytes, _ := xml.Marshal(out)
	c.Header("Content-Type", "application/xml;charset=utf-8")
	c.String(http.StatusOK, string(xmlBytes))
}
func (h *Handler) WxSetMenu(c *gin.Context) { c.JSON(http.StatusOK, common.Success("ok")) }

func (h *Handler) mustLoginUser(c *gin.Context) *entity.User {
	uidAny, ok := c.Get("loginUserId")
	if !ok {
		panic(common.NewBizError(common.NotLoginError, "未登录"))
	}
	uid, err := toInt64(uidAny)
	mustNoErr(err)
	u, err := h.userSvc.GetByID(uid)
	mustNoErr(err)
	if u == nil {
		panic(common.NewBizError(common.NotLoginError, "未登录"))
	}
	if strings.EqualFold(strings.TrimSpace(u.UserRole), common.BanRole) {
		panic(common.NewBizError(common.ForbiddenError, "账号已被禁用"))
	}
	return u
}
func (h *Handler) mustAdmin(c *gin.Context) {
	if !h.userSvc.IsAdmin(h.mustLoginUser(c)) {
		panic(common.NewBizError(common.NoAuthError, "无权限"))
	}
}

func (h *Handler) mustSuperAdmin(c *gin.Context) {
	if !h.userSvc.IsSuperAdmin(h.mustLoginUser(c)) {
		panic(common.NewBizError(common.NoAuthError, "仅超级管理员可操作"))
	}
}

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
	acceptedBefore := int64(-1)
	if strings.EqualFold(result.Message, "Accepted") {
		status = submitStatusAccepted
		if err := h.db.Model(&entity.QuestionSubmit{}).
			Where("userId = ? AND questionId = ? AND status = ? AND isDelete = 0", s.UserID, s.QuestionID, submitStatusAccepted).
			Count(&acceptedBefore).Error; err != nil {
			acceptedBefore = -1
		}
	}
	jInfo, _ := json.Marshal(result)
	if err := h.finishJudgeSubmit(submitID, status, string(jInfo)); err != nil {
		return err
	}
	if status == submitStatusAccepted {
		_ = h.db.Model(&entity.Question{}).Where("id=?", q.ID).Update("acceptedNum", gorm.Expr("acceptedNum + 1")).Error
		if acceptedBefore == 0 {
			h.recordFirstAccepted24h(s.UserID, s.QuestionID, s.ID)
		}
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

func (h *Handler) finishJudgeSubmit(submitID int64, status int32, judgeInfo string) error {
	res := h.db.Model(&entity.QuestionSubmit{}).
		Where("id=? AND status=? AND isDelete=0", submitID, submitStatusRunning).
		Updates(map[string]interface{}{"status": status, "judgeInfo": judgeInfo, "updateTime": time.Now()})
	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (h *Handler) questionList(c *gin.Context, onlyMine bool, raw bool, mine ...int64) {
	var req struct {
		common.PageRequest
		UserID     int64    `json:"userId"`
		Title      string   `json:"title"`
		SearchText string   `json:"searchText"`
		Tags       []string `json:"tags"`
	}
	mustBindJSON(c, &req)
	req.Normalize()
	if req.PageSize > 20 && !raw {
		panic(common.NewBizError(common.ParamsError, "请求参数错误"))
	}
	var list []entity.Question
	query := h.db.Model(&entity.Question{}).Where("isDelete=0")
	if onlyMine {
		query = query.Where("userId=?", mine[0])
	} else if req.UserID > 0 {
		query = query.Where("userId=?", req.UserID)
	}
	if req.Title != "" {
		query = query.Where("title like ?", "%"+req.Title+"%")
	}
	if req.SearchText != "" {
		query = query.Where("(title like ? OR content like ?)", "%"+req.SearchText+"%", "%"+req.SearchText+"%")
	}
	for _, tag := range req.Tags {
		query = query.Where("tags like ?", "%\""+tag+"\"%")
	}
	var total int64
	mustNoErr(query.Count(&total).Error)
	mustNoErr(query.Offset(int((req.Current - 1) * req.PageSize)).Limit(int(req.PageSize)).Order("id desc").Find(&list).Error)
	if raw {
		c.JSON(http.StatusOK, common.Success(common.PageResult{Records: list, Total: total, Current: req.Current, Size: req.PageSize}))
		return
	}
	vos := make([]map[string]interface{}, 0, len(list))
	for i := range list {
		vos = append(vos, questionVO(&list[i]))
	}
	c.JSON(http.StatusOK, common.Success(common.PageResult{Records: vos, Total: total, Current: req.Current, Size: req.PageSize}))
}
func (h *Handler) postList(c *gin.Context, mine bool) {
	var req struct {
		common.PageRequest
		UserID     int64    `json:"userId"`
		Title      string   `json:"title"`
		Content    string   `json:"content"`
		SearchText string   `json:"searchText"`
		Tags       []string `json:"tags"`
	}
	mustBindJSON(c, &req)
	req.Normalize()
	if req.PageSize > 20 {
		panic(common.NewBizError(common.ParamsError, "请求参数错误"))
	}
	if !mine && h.cfg.ES.Enabled {
		if req.PageSize > 0 {
			ids, total, err := h.searchPostIDsFromES(req.Current, req.PageSize, req.UserID, req.Title, req.Content, req.SearchText, req.Tags)
			if err == nil && len(ids) > 0 {
				var posts []entity.Post
				mustNoErr(h.db.Where("id in ? AND isDelete=0", ids).Find(&posts).Error)
				orderMap := map[int64]int{}
				for i, id := range ids {
					orderMap[id] = i
				}
				sort.Slice(posts, func(i, j int) bool { return orderMap[posts[i].ID] < orderMap[posts[j].ID] })
				vos := make([]map[string]interface{}, 0, len(posts))
				for i := range posts {
					vos = append(vos, h.postVO(c, &posts[i]))
				}
				c.JSON(http.StatusOK, common.Success(common.PageResult{Records: vos, Total: total, Current: req.Current, Size: req.PageSize}))
				return
			}
		}
	}
	query := h.db.Model(&entity.Post{}).Where("isDelete=0")
	if req.Title != "" {
		query = query.Where("title like ?", "%"+req.Title+"%")
	}
	if req.Content != "" {
		query = query.Where("content like ?", "%"+req.Content+"%")
	}
	if req.SearchText != "" {
		query = query.Where("(title like ? OR content like ?)", "%"+req.SearchText+"%", "%"+req.SearchText+"%")
	}
	for _, tag := range req.Tags {
		query = query.Where("tags like ?", "%\""+tag+"\"%")
	}
	if mine {
		query = query.Where("userId=?", h.mustLoginUser(c).ID)
	} else if req.UserID > 0 {
		query = query.Where("userId=?", req.UserID)
	}
	var total int64
	mustNoErr(query.Count(&total).Error)
	var list []entity.Post
	mustNoErr(query.Offset(int((req.Current - 1) * req.PageSize)).Limit(int(req.PageSize)).Order("id desc").Find(&list).Error)
	vos := make([]map[string]interface{}, 0, len(list))
	for i := range list {
		vos = append(vos, h.postVO(c, &list[i]))
	}
	c.JSON(http.StatusOK, common.Success(common.PageResult{Records: vos, Total: total, Current: req.Current, Size: req.PageSize}))
}
func (h *Handler) listFavourByUserID(c *gin.Context, uid int64) {
	var req common.PageRequest
	mustBindJSON(c, &req)
	h.listFavourByUserIDWithReq(c, uid, req)
}
func (h *Handler) listFavourByUserIDWithReq(c *gin.Context, uid int64, req common.PageRequest) {
	req.Normalize()
	var total int64
	mustNoErr(h.db.Model(&entity.PostFavour{}).Where("userId=?", uid).Count(&total).Error)
	var favs []entity.PostFavour
	mustNoErr(h.db.Where("userId=?", uid).Offset(int((req.Current - 1) * req.PageSize)).Limit(int(req.PageSize)).Order("id desc").Find(&favs).Error)
	postIDs := make([]int64, 0, len(favs))
	for _, f := range favs {
		postIDs = append(postIDs, f.PostID)
	}
	var posts []entity.Post
	if len(postIDs) > 0 {
		mustNoErr(h.db.Where("id in ? AND isDelete=0", postIDs).Find(&posts).Error)
	}
	vos := make([]map[string]interface{}, 0, len(posts))
	for i := range posts {
		vos = append(vos, h.postVO(c, &posts[i]))
	}
	c.JSON(http.StatusOK, common.Success(common.PageResult{Records: vos, Total: total, Current: req.Current, Size: req.PageSize}))
}
func (h *Handler) postVO(c *gin.Context, p *entity.Post) map[string]interface{} {
	var tags []string
	_ = json.Unmarshal([]byte(p.Tags), &tags)
	var user entity.User
	_ = h.db.Where("id=? AND isDelete=0", p.UserID).First(&user).Error
	hasThumb, hasFavour := false, false
	login := h.loginUserOrNil(c)
	if login != nil {
		var t entity.PostThumb
		var f entity.PostFavour
		hasThumb = h.db.Where("postId=? AND userId=?", p.ID, login.ID).First(&t).Error == nil
		hasFavour = h.db.Where("postId=? AND userId=?", p.ID, login.ID).First(&f).Error == nil
	}
	return map[string]interface{}{"id": p.ID, "title": p.Title, "content": p.Content, "thumbNum": p.ThumbNum, "favourNum": p.FavourNum, "userId": p.UserID, "createTime": p.CreateTime, "updateTime": p.UpdateTime, "tagList": tags, "user": userVO(&user), "hasThumb": hasThumb, "hasFavour": hasFavour}
}
func questionVO(q *entity.Question) map[string]interface{} {
	var tags []string
	var cfg map[string]interface{}
	var sampleCases []judgeCase
	_ = json.Unmarshal([]byte(q.Tags), &tags)
	_ = json.Unmarshal([]byte(q.JudgeConfig), &cfg)
	_ = json.Unmarshal([]byte(q.SampleCase), &sampleCases)
	return map[string]interface{}{"id": q.ID, "title": q.Title, "content": q.Content, "tags": tags, "submitNum": q.SubmitNum, "acceptedNum": q.AcceptedNum, "judgeConfig": cfg, "sampleCase": sampleCases, "thumbNum": q.ThumbNum, "favourNum": q.FavourNum, "userId": q.UserID, "createTime": q.CreateTime, "updateTime": q.UpdateTime}
}
func userVO(u *entity.User) map[string]interface{} {
	return map[string]interface{}{"id": u.ID, "userName": u.UserName, "userAvatar": u.UserAvatar, "userProfile": u.UserProfile, "userRole": u.UserRole, "createTime": u.CreateTime}
}
func loginUserVO(u *entity.User) map[string]interface{} {
	return map[string]interface{}{"id": u.ID, "userName": u.UserName, "userAvatar": u.UserAvatar, "userProfile": u.UserProfile, "userRole": u.UserRole, "createTime": u.CreateTime, "updateTime": u.UpdateTime}
}
func (h *Handler) loginUserOrNil(c *gin.Context) *entity.User {
	sess := sessions.Default(c)
	uid := sess.Get(common.UserLoginState)
	if uid == nil {
		return nil
	}
	id, err := toInt64(uid)
	if err != nil {
		return nil
	}
	u, err := h.userSvc.GetByID(id)
	if err != nil {
		return nil
	}
	return u
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

func mustBindJSON(c *gin.Context, v interface{}) {
	if c.ShouldBindJSON(v) != nil {
		panic(common.NewBizError(common.ParamsError, "请求参数错误"))
	}
}
func mustNoErr(err error) {
	if err != nil {
		panic(err)
	}
}
func parseIDQuery(c *gin.Context) int64 {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Query("id")), 10, 64)
	if err != nil || id <= 0 {
		panic(common.NewBizError(common.ParamsError, "请求参数错误"))
	}
	return id
}
func toInt64(v interface{}) (int64, error) {
	switch t := v.(type) {
	case int64:
		return t, nil
	case int:
		return int64(t), nil
	case int32:
		return int64(t), nil
	case float64:
		return int64(t), nil
	case string:
		return strconv.ParseInt(t, 10, 64)
	default:
		return 0, errors.New("invalid type")
	}
}

func validQuestionInput(q *entity.Question, add bool) {
	if add && (strings.TrimSpace(q.Title) == "" || strings.TrimSpace(q.Content) == "" || strings.TrimSpace(q.Tags) == "") {
		panic(common.NewBizError(common.ParamsError, "请求参数错误"))
	}
	if len(q.Title) > 80 || len(q.Content) > 8192 || len(q.Answer) > 8192 || len(q.SampleCase) > 8192 || len(q.JudgeCase) > 65535 || len(q.JudgeConfig) > 8192 {
		panic(common.NewBizError(common.ParamsError, "请求参数错误"))
	}
}

func validPostInput(p *entity.Post, add bool) {
	if add && (strings.TrimSpace(p.Title) == "" || strings.TrimSpace(p.Content) == "" || strings.TrimSpace(p.Tags) == "") {
		panic(common.NewBizError(common.ParamsError, "请求参数错误"))
	}
	if len(p.Title) > 80 || len(p.Content) > 8192 {
		panic(common.NewBizError(common.ParamsError, "请求参数错误"))
	}
}

func validUploadFile(size int64, filename, biz string) {
	if biz != "user_avatar" {
		panic(common.NewBizError(common.ParamsError, "请求参数错误"))
	}
	if size > 1024*1024 {
		panic(common.NewBizError(common.ParamsError, "文件大小不能超过 1M"))
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
	allowed := map[string]bool{"jpeg": true, "jpg": true, "svg": true, "png": true, "webp": true}
	if !allowed[ext] {
		panic(common.NewBizError(common.ParamsError, "文件类型错误"))
	}
}

type judgeCase struct {
	Input  string `json:"input"`
	Output string `json:"output"`
}

type judgeInfo struct {
	Message string `json:"message"`
	Memory  int64  `json:"memory"`
	Time    int64  `json:"time"`
	Score   int64  `json:"score"`
}

type executeCodeResp struct {
	OutputList []string  `json:"outputList"`
	JudgeInfo  judgeInfo `json:"judgeInfo"`
}

func (h *Handler) executeCode(language, code string, inputList []string) (*executeCodeResp, error) {
	if h.cfg.Judge.SandboxType == "remote" && h.cfg.Judge.SandboxURL != "" {
		reqBody, _ := json.Marshal(map[string]interface{}{
			"language":  language,
			"code":      code,
			"inputList": inputList,
		})
		req, _ := http.NewRequest(http.MethodPost, h.cfg.Judge.SandboxURL, bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		if strings.TrimSpace(h.cfg.Judge.AuthSecret) != "" {
			req.Header.Set("auth", h.cfg.Judge.AuthSecret)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		var r executeCodeResp
		if err = json.Unmarshal(b, &r); err != nil {
			return nil, err
		}
		return &r, nil
	}
	return runLocalCode(language, code, inputList)
}

func runLocalCode(language, code string, inputList []string) (*executeCodeResp, error) {
	lang := strings.ToLower(strings.TrimSpace(language))
	switch lang {
	case "go":
		return runLocalGo(code, inputList)
	default:
		return nil, fmt.Errorf("unsupported language in local sandbox: %s", language)
	}
}

func runLocalGo(code string, inputList []string) (*executeCodeResp, error) {
	root, err := os.MkdirTemp("", "oj-go-run-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(root)
	mainFile := filepath.Join(root, "main.go")
	if err = os.WriteFile(mainFile, []byte(code), 0o644); err != nil {
		return nil, err
	}
	ins := inputList
	if len(ins) == 0 {
		ins = []string{""}
	}
	outputs := make([]string, 0, len(ins))
	maxTime := int64(0)
	for _, in := range ins {
		start := time.Now()
		runCtx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		goBin := "go"
		if _, statErr := os.Stat("/usr/local/go/bin/go"); statErr == nil {
			goBin = "/usr/local/go/bin/go"
		}
		cmd := exec.CommandContext(runCtx, goBin, "run", "main.go")
		cmd.Dir = root
		cmd.Stdin = strings.NewReader(in)
		out, runErr := cmd.CombinedOutput()
		cancel()
		cost := time.Since(start).Milliseconds()
		if cost > maxTime {
			maxTime = cost
		}
		if runErr != nil {
			if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
				return nil, fmt.Errorf("time limit exceeded")
			}
			msg := strings.TrimSpace(string(out))
			if msg == "" {
				msg = runErr.Error()
			}
			if strings.Contains(msg, "syntax error") || strings.Contains(msg, "undefined") {
				return nil, fmt.Errorf("compile error: %s", msg)
			}
			return nil, fmt.Errorf("runtime error: %s", msg)
		}
		outputs = append(outputs, strings.TrimSpace(string(out)))
	}
	return &executeCodeResp{
		OutputList: outputs,
		JudgeInfo: judgeInfo{
			Message: "OK",
			Memory:  0,
			Time:    maxTime,
		},
	}, nil
}

func doJudge(cases []judgeCase, outputs []string, sandbox judgeInfo, judgeConfigJSON string) judgeInfo {
	msg := strings.TrimSpace(strings.ToLower(sandbox.Message))
	if msg != "" && msg != "ok" && msg != "accepted" {
		if strings.Contains(msg, "compile") {
			return judgeInfo{Message: "Compile Error", Memory: sandbox.Memory, Time: sandbox.Time, Score: 0}
		}
		if strings.Contains(msg, "runtime") || strings.Contains(msg, "panic") || strings.Contains(msg, "exception") {
			return judgeInfo{Message: "Runtime Error", Memory: sandbox.Memory, Time: sandbox.Time, Score: 0}
		}
		if strings.Contains(msg, "time") {
			return judgeInfo{Message: "Time Limit Exceeded", Memory: sandbox.Memory, Time: sandbox.Time, Score: 0}
		}
		if strings.Contains(msg, "memory") {
			return judgeInfo{Message: "Memory Limit Exceeded", Memory: sandbox.Memory, Time: sandbox.Time, Score: 0}
		}
		return judgeInfo{Message: sandbox.Message, Memory: sandbox.Memory, Time: sandbox.Time, Score: 0}
	}

	passed := int64(0)
	total := int64(len(cases))
	for i, c := range cases {
		got := ""
		if i < len(outputs) {
			got = strings.TrimSpace(outputs[i])
		}
		if strings.TrimSpace(c.Output) == got {
			passed++
		}
	}
	cfg := map[string]int64{}
	_ = json.Unmarshal([]byte(judgeConfigJSON), &cfg)
	if m, ok := cfg["memoryLimit"]; ok && m > 0 && sandbox.Memory > m {
		return judgeInfo{Message: "Memory Limit Exceeded", Memory: sandbox.Memory, Time: sandbox.Time, Score: 0}
	}
	if t, ok := cfg["timeLimit"]; ok && t > 0 && sandbox.Time > t {
		return judgeInfo{Message: "Time Limit Exceeded", Memory: sandbox.Memory, Time: sandbox.Time, Score: 0}
	}
	if total == 0 {
		return judgeInfo{Message: "Accepted", Memory: sandbox.Memory, Time: sandbox.Time, Score: 100}
	}
	score := passed * 100 / total
	if passed == total {
		return judgeInfo{Message: "Accepted", Memory: sandbox.Memory, Time: sandbox.Time, Score: 100}
	}
	if passed > 0 {
		return judgeInfo{Message: "Partially Correct", Memory: sandbox.Memory, Time: sandbox.Time, Score: score}
	}
	return judgeInfo{Message: "Wrong Answer", Memory: sandbox.Memory, Time: sandbox.Time, Score: 0}
}

func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

type cdata string

func (c cdata) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.EncodeElement(struct {
		Text string `xml:",cdata"`
	}{Text: string(c)}, start)
}

type wxInMessage struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType"`
	Content      string   `xml:"Content"`
	Event        string   `xml:"Event"`
	EventKey     string   `xml:"EventKey"`
}

type wxOutMessage struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   cdata    `xml:"ToUserName"`
	FromUserName cdata    `xml:"FromUserName"`
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      cdata    `xml:"MsgType"`
	Content      cdata    `xml:"Content"`
}

func wxCheckSignature(token, signature, timestamp, nonce string) bool {
	arr := []string{token, timestamp, nonce}
	sort.Strings(arr)
	sum := sha1.Sum([]byte(strings.Join(arr, "")))
	return fmt.Sprintf("%x", sum) == signature
}

func (h *Handler) fetchWxOpenUserInfo(code string) (unionID, openID string, err error) {
	tokenURL := fmt.Sprintf("https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code", h.cfg.WX.OpenAppID, h.cfg.WX.OpenSecret, code)
	resp, err := http.Get(tokenURL)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var tk map[string]interface{}
	if json.Unmarshal(body, &tk) != nil {
		return "", "", errors.New("wx token parse error")
	}
	accessToken, _ := tk["access_token"].(string)
	openID, _ = tk["openid"].(string)
	unionID, _ = tk["unionid"].(string)
	if accessToken == "" || openID == "" {
		return unionID, openID, errors.New("wx token missing")
	}
	infoURL := fmt.Sprintf("https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s", accessToken, openID)
	resp2, err := http.Get(infoURL)
	if err != nil {
		return unionID, openID, err
	}
	defer resp2.Body.Close()
	body2, _ := io.ReadAll(resp2.Body)
	var info map[string]interface{}
	if json.Unmarshal(body2, &info) == nil {
		if u, ok := info["unionid"].(string); ok && u != "" {
			unionID = u
		}
		if o, ok := info["openid"].(string); ok && o != "" {
			openID = o
		}
	}
	return unionID, openID, nil
}

func (h *Handler) uploadToCOS(c *gin.Context, fileHeader *multipart.FileHeader, key string) (string, error) {
	u, err := url.Parse(h.cfg.COS.BucketURL)
	if err != nil {
		return "", err
	}
	b := &cos.BaseURL{BucketURL: u}
	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  h.cfg.COS.SecretID,
			SecretKey: h.cfg.COS.SecretKey,
		},
	})
	f, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()
	_, err = client.Object.Put(c.Request.Context(), key, f, nil)
	if err != nil {
		return "", err
	}
	host := strings.TrimRight(h.cfg.COS.Host, "/")
	if host != "" {
		return host + key, nil
	}
	return h.cfg.COS.BucketURL + key, nil
}

func (h *Handler) searchPostIDsFromES(current, pageSize, userID int64, title, content, searchText string, tags []string) ([]int64, int64, error) {
	from := (current - 1) * pageSize
	query := map[string]interface{}{
		"from": from,
		"size": pageSize,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": []interface{}{map[string]interface{}{"term": map[string]interface{}{"isDelete": 0}}},
				"must":   []interface{}{},
			},
		},
	}
	boolQ := query["query"].(map[string]interface{})["bool"].(map[string]interface{})
	filter := boolQ["filter"].([]interface{})
	must := boolQ["must"].([]interface{})
	if userID > 0 {
		filter = append(filter, map[string]interface{}{"term": map[string]interface{}{"userId": userID}})
	}
	if title != "" {
		must = append(must, map[string]interface{}{"match": map[string]interface{}{"title": title}})
	}
	if content != "" {
		must = append(must, map[string]interface{}{"match": map[string]interface{}{"content": content}})
	}
	if searchText != "" {
		must = append(must, map[string]interface{}{"multi_match": map[string]interface{}{"query": searchText, "fields": []string{"title", "content", "description"}}})
	}
	for _, tag := range tags {
		filter = append(filter, map[string]interface{}{"term": map[string]interface{}{"tags": tag}})
	}
	boolQ["filter"] = filter
	boolQ["must"] = must
	body, _ := json.Marshal(query)
	url := strings.TrimRight(h.cfg.ES.URL, "/") + "/" + h.cfg.ES.Index + "/_search"
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err = json.Unmarshal(respBody, &result); err != nil {
		return nil, 0, err
	}
	hitsObj, ok := result["hits"].(map[string]interface{})
	if !ok {
		return nil, 0, errors.New("invalid es hits")
	}
	total := int64(0)
	if totalObj, ok := hitsObj["total"].(map[string]interface{}); ok {
		if v, ok := totalObj["value"].(float64); ok {
			total = int64(v)
		}
	}
	rawHits, ok := hitsObj["hits"].([]interface{})
	if !ok {
		return nil, total, nil
	}
	ids := make([]int64, 0, len(rawHits))
	for _, h1 := range rawHits {
		m, ok := h1.(map[string]interface{})
		if !ok {
			continue
		}
		src, _ := m["_source"].(map[string]interface{})
		if src != nil {
			if v, ok := src["id"].(float64); ok {
				ids = append(ids, int64(v))
				continue
			}
		}
		if sid, ok := m["_id"].(string); ok {
			if id, err := strconv.ParseInt(sid, 10, 64); err == nil {
				ids = append(ids, id)
			}
		}
	}
	return ids, total, nil
}

func (h *Handler) AgentGenerateQuestionSolution(c *gin.Context) {
	login := h.mustLoginUser(c)
	h.mustAdmin(c)
	if !h.cfg.Agent.Enabled {
		panic(common.NewBizError(common.OperationError, "Agent 功能未启用"))
	}
	var req struct {
		QuestionID int64 `json:"questionId"`
	}
	mustBindJSON(c, &req)
	if req.QuestionID <= 0 {
		panic(common.NewBizError(common.ParamsError, "请求参数错误"))
	}
	var q entity.Question
	mustNoErr(h.db.Where("id=? AND isDelete=0", req.QuestionID).First(&q).Error)
	task := entity.AgentSolutionTask{
		QuestionID:    req.QuestionID,
		TriggerUserID: login.ID,
		Status:        "pending",
		ModelName:     strings.TrimSpace(h.cfg.Agent.Model),
	}
	mustNoErr(h.db.Create(&task).Error)
	go h.runAgentSolutionTask(task.ID)
	c.JSON(http.StatusOK, common.Success(map[string]interface{}{
		"taskId": task.ID,
		"status": task.Status,
	}))
}

func (h *Handler) AgentGetSolutionTask(c *gin.Context) {
	h.mustAdmin(c)
	id := parseIDQuery(c)
	var task entity.AgentSolutionTask
	mustNoErr(h.db.Where("id = ? AND isDelete = 0", id).First(&task).Error)
	c.JSON(http.StatusOK, common.Success(task))
}

func (h *Handler) AgentListSolutionTask(c *gin.Context) {
	h.mustAdmin(c)
	var req struct {
		common.PageRequest
		QuestionID int64  `json:"questionId"`
		Status     string `json:"status"`
	}
	mustBindJSON(c, &req)
	req.Normalize()
	if req.PageSize > 20 {
		panic(common.NewBizError(common.ParamsError, "请求参数错误"))
	}
	query := h.db.Model(&entity.AgentSolutionTask{}).Where("isDelete = 0")
	if req.QuestionID > 0 {
		query = query.Where("questionId = ?", req.QuestionID)
	}
	if strings.TrimSpace(req.Status) != "" {
		query = query.Where("status = ?", strings.TrimSpace(req.Status))
	}
	var total int64
	mustNoErr(query.Count(&total).Error)
	var list []entity.AgentSolutionTask
	mustNoErr(query.Order("id desc").Offset(int((req.Current - 1) * req.PageSize)).Limit(int(req.PageSize)).Find(&list).Error)
	c.JSON(http.StatusOK, common.Success(common.PageResult{
		Records: list,
		Total:   total,
		Current: req.Current,
		Size:    req.PageSize,
	}))
}

func (h *Handler) runAgentSolutionTask(taskID int64) {
	var task entity.AgentSolutionTask
	if err := h.db.Where("id = ? AND isDelete = 0", taskID).First(&task).Error; err != nil {
		return
	}
	_ = h.db.Model(&entity.AgentSolutionTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
		"status": "running",
	}).Error

	var q entity.Question
	if err := h.db.Where("id = ? AND isDelete = 0", task.QuestionID).First(&q).Error; err != nil {
		h.failAgentTask(taskID, "题目不存在")
		return
	}
	agentUserID, err := h.ensureOJAgentUser()
	if err != nil {
		h.failAgentTask(taskID, "初始化 AI 账号失败: "+err.Error())
		return
	}

	maxRound := h.cfg.Agent.MaxRound
	if maxRound <= 0 {
		maxRound = 3
	}
	type sampleCase struct {
		Input  string `json:"input"`
		Output string `json:"output"`
	}
	var sampleCases []sampleCase
	_ = json.Unmarshal([]byte(q.SampleCase), &sampleCases)

	prompt := h.buildAgentPrompt(q, sampleCases)
	feedback := ""
	var finalMarkdown string
	var finalCode string
	published := false
	for i := 1; i <= maxRound; i++ {
		reply, err := h.callAgentLLM(prompt, feedback)
		if err != nil {
			h.failAgentTask(taskID, "调用模型失败: "+err.Error())
			return
		}
		title, summary, markdown, code := parseAgentReply(reply, q.Title)
		if strings.TrimSpace(code) == "" {
			feedback = "你没有提供可运行的 Go 代码，请输出完整可编译的 Go 代码。"
			continue
		}
		submitID, judgeMessage, judgeScore, err := h.submitAgentCodeAndJudge(agentUserID, q.ID, code)
		if err != nil {
			feedback = "提交评测失败：" + err.Error() + "。请修复后重试。"
			continue
		}
		finalCode = code
		finalMarkdown = markdown
		_ = h.db.Model(&entity.AgentSolutionTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
			"attempts":       i,
			"answerCode":     code,
			"answerMarkdown": markdown,
			"lastError":      fmt.Sprintf("latest submit id=%d", submitID),
		}).Error
		if strings.EqualFold(judgeMessage, "Accepted") {
			postID, postURL, err := h.publishAgentBlog(q.ID, title, summary, markdown)
			if err != nil {
				h.failAgentTask(taskID, "发布博客失败: "+err.Error())
				return
			}
			_ = h.db.Create(&entity.QuestionSolutionPost{
				QuestionID: q.ID,
				PostID:     postID,
				UserID:     agentUserID,
			}).Error
			_ = h.db.Model(&entity.AgentSolutionTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
				"status":      "success",
				"blogPostId":  postID,
				"blogPostUrl": postURL,
				"attempts":    i,
				"lastError":   "",
			}).Error
			published = true
			break
		}
		feedback = "判题结果: " + judgeMessage + fmt.Sprintf("，得分=%d。请继续优化代码并输出完整题解与代码。", judgeScore)
	}
	if !published {
		if strings.TrimSpace(finalCode) == "" && strings.TrimSpace(finalMarkdown) == "" {
			h.failAgentTask(taskID, "Agent 未生成有效内容")
			return
		}
		h.failAgentTask(taskID, "达到最大迭代次数，仍未通过评测")
	}
}

func (h *Handler) ensureOJAgentUser() (int64, error) {
	account := strings.TrimSpace(h.cfg.Blog.AgentAccount)
	if account == "" {
		account = "kkg_agent"
	}
	var u entity.User
	err := h.db.Where("userAccount = ? AND isDelete = 0", account).First(&u).Error
	if err == nil {
		if strings.EqualFold(u.UserRole, common.BanRole) {
			_ = h.db.Model(&entity.User{}).Where("id = ?", u.ID).Update("userRole", common.DefaultRole).Error
			u.UserRole = common.DefaultRole
		}
		return u.ID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	password := strings.TrimSpace(h.cfg.Blog.AgentPassword)
	if len(password) < 8 {
		return 0, errors.New("invalid config: blog.agent_password must be at least 8 characters")
	}
	u = entity.User{
		UserAccount:  account,
		UserPassword: hashAgentPassword(password),
		UserName:     strings.TrimSpace(h.cfg.Blog.AgentDisplayName),
		UserRole:     common.DefaultRole,
	}
	if strings.TrimSpace(u.UserName) == "" {
		u.UserName = "KKG Agent"
	}
	if err := h.db.Create(&u).Error; err != nil {
		return 0, err
	}
	return u.ID, nil
}

func (h *Handler) submitAgentCodeAndJudge(userID, questionID int64, code string) (int64, string, int64, error) {
	qs := entity.QuestionSubmit{
		Language:   "go",
		Code:       code,
		QuestionID: questionID,
		UserID:     userID,
		Status:     0,
		JudgeInfo:  "{}",
	}
	if err := h.db.Create(&qs).Error; err != nil {
		return 0, "", 0, err
	}
	_ = h.db.Model(&entity.Question{}).Where("id=?", questionID).Update("submitNum", gorm.Expr("submitNum + 1")).Error
	h.judgeAsync(qs.ID)
	var out entity.QuestionSubmit
	if err := h.db.Where("id = ? AND isDelete = 0", qs.ID).First(&out).Error; err != nil {
		return qs.ID, "", 0, err
	}
	var info struct {
		Message string `json:"message"`
		Score   int64  `json:"score"`
	}
	_ = json.Unmarshal([]byte(out.JudgeInfo), &info)
	msg := strings.TrimSpace(info.Message)
	if msg == "" {
		if out.Status == 2 {
			msg = "Accepted"
		} else if out.Status == 3 {
			msg = "Wrong Answer"
		} else {
			msg = "Unknown"
		}
	}
	return qs.ID, msg, info.Score, nil
}

func (h *Handler) failAgentTask(taskID int64, msg string) {
	_ = h.db.Model(&entity.AgentSolutionTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
		"status":    "failed",
		"lastError": msg,
	}).Error
}

func (h *Handler) buildAgentPrompt(q entity.Question, sampleCases interface{}) string {
	sb := strings.Builder{}
	sb.WriteString("你是资深算法工程师，请为以下题目生成高质量题解。")
	sb.WriteString("\n要求：")
	sb.WriteString("\n1) 明确知识点")
	sb.WriteString("\n2) 说明核心思路与复杂度")
	sb.WriteString("\n3) 提供可通过的 Go 代码")
	sb.WriteString("\n4) 使用 Markdown 输出，结构清晰")
	sb.WriteString("\n输出必须是 JSON，字段: title, summary, markdown, code。")
	sb.WriteString("\n题目标题：")
	sb.WriteString(q.Title)
	sb.WriteString("\n题目描述：\n")
	sb.WriteString(q.Content)
	sb.WriteString("\n样例（仅供理解）：\n")
	raw, _ := json.Marshal(sampleCases)
	sb.Write(raw)
	return sb.String()
}

func (h *Handler) callAgentLLM(prompt, feedback string) (string, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(h.cfg.Agent.BaseURL), "/")
	if baseURL == "" || strings.TrimSpace(h.cfg.Agent.APIKey) == "" || strings.TrimSpace(h.cfg.Agent.Model) == "" {
		return "", errors.New("agent 配置不完整")
	}
	userPrompt := prompt
	if strings.TrimSpace(feedback) != "" {
		userPrompt += "\n\n上一轮反馈：\n" + feedback
	}
	payload := map[string]interface{}{
		"model": h.cfg.Agent.Model,
		"messages": []map[string]string{
			{"role": "system", "content": "你是 OJ 题解生成助手。"},
			{"role": "user", "content": userPrompt},
		},
		"temperature": h.cfg.Agent.Temperature,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.cfg.Agent.APIKey)
	cli := &http.Client{Timeout: 90 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", errors.New("模型服务错误: " + string(data))
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", errors.New("模型返回为空")
	}
	return strings.TrimSpace(out.Choices[0].Message.Content), nil
}

func parseAgentReply(reply string, questionTitle string) (title, summary, markdown, code string) {
	title = "题解：" + strings.TrimSpace(questionTitle)
	summary = "由 Agent 自动生成的题解"
	markdown = strings.TrimSpace(reply)
	code = ""
	var obj struct {
		Title    string `json:"title"`
		Summary  string `json:"summary"`
		Markdown string `json:"markdown"`
		Code     string `json:"code"`
	}
	if err := json.Unmarshal([]byte(reply), &obj); err == nil {
		if strings.TrimSpace(obj.Title) != "" {
			title = strings.TrimSpace(obj.Title)
		}
		if strings.TrimSpace(obj.Summary) != "" {
			summary = strings.TrimSpace(obj.Summary)
		}
		if strings.TrimSpace(obj.Markdown) != "" {
			markdown = strings.TrimSpace(obj.Markdown)
		}
		if strings.TrimSpace(obj.Code) != "" {
			code = strings.TrimSpace(obj.Code)
		}
	}
	if code == "" {
		start := strings.Index(markdown, "```go")
		if start >= 0 {
			rest := markdown[start+5:]
			end := strings.Index(rest, "```")
			if end > 0 {
				code = strings.TrimSpace(rest[:end])
			}
		}
	}
	return
}

func (h *Handler) publishAgentBlog(questionID int64, title, summary, markdown string) (int64, string, error) {
	base := strings.TrimRight(strings.TrimSpace(h.cfg.Blog.BaseURL), "/")
	if base == "" {
		return 0, "", errors.New("blog base_url 未配置")
	}
	account := strings.TrimSpace(h.cfg.Blog.AgentAccount)
	password := strings.TrimSpace(h.cfg.Blog.AgentPassword)
	email := strings.TrimSpace(h.cfg.Blog.AgentEmail)
	if account == "" || password == "" || email == "" {
		return 0, "", errors.New("blog agent 账号配置不完整")
	}

	token, err := h.blogLoginOrRegister(base, account, password, email)
	if err != nil {
		return 0, "", err
	}

	content := markdown + fmt.Sprintf("\n\n---\n\n> 该题解由 KKG Agent 自动生成并经评测通过后发布。题号：%d", questionID) +
		fmt.Sprintf("\n\n👉 [跳转到题目](/oj/questions/%d)", questionID)
	createBody := map[string]interface{}{
		"title":       title,
		"summary":     summary,
		"tags":        []string{"题解", "Agent", "OJ", fmt.Sprintf("Q%d", questionID)},
		"raw_content": content,
	}
	postResp, err := h.blogRequest(base+"/api/v1/posts", token, createBody)
	if err != nil {
		return 0, "", err
	}
	postID, _ := toInt64(postResp["id"])
	if postID <= 0 {
		return 0, "", errors.New("博客创建失败: 无 post id")
	}
	_, err = h.blogRequest(fmt.Sprintf("%s/api/v1/posts/%d/publish", base, postID), token, map[string]interface{}{})
	if err != nil {
		return 0, "", err
	}
	postURL := fmt.Sprintf("/posts/%d", postID)
	return postID, postURL, nil
}

func (h *Handler) blogLoginOrRegister(base, account, password, email string) (string, error) {
	loginResp, err := h.blogRequest(base+"/api/v1/auth/login", "", map[string]interface{}{
		"account":  account,
		"password": password,
	})
	if err == nil {
		token, _ := loginResp["access_token"].(string)
		if strings.TrimSpace(token) != "" {
			return token, nil
		}
	}
	_, _ = h.blogRequest(base+"/api/v1/auth/register", "", map[string]interface{}{
		"username": account,
		"email":    email,
		"password": password,
	})
	loginResp, err = h.blogRequest(base+"/api/v1/auth/login", "", map[string]interface{}{
		"account":  account,
		"password": password,
	})
	if err != nil {
		return "", err
	}
	token, _ := loginResp["access_token"].(string)
	if strings.TrimSpace(token) == "" {
		return "", errors.New("blog 登录成功但 token 为空")
	}
	return token, nil
}

func (h *Handler) blogRequest(url string, token string, body map[string]interface{}) (map[string]interface{}, error) {
	payload, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	cli := &http.Client{Timeout: 30 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out struct {
		Code    int                    `json:"code"`
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, errors.New("blog 响应解析失败: " + string(raw))
	}
	if resp.StatusCode >= 300 || out.Code != 0 {
		return nil, errors.New(out.Message)
	}
	return out.Data, nil
}

func hashAgentPassword(password string) string {
	sum := md5.Sum([]byte(agentPasswordSalt + password))
	return hex.EncodeToString(sum[:])
}
