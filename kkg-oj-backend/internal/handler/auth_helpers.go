package handler

import (
	"errors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"path/filepath"
	"strconv"
	"strings"
	"yuoj-go-backend/internal/common"
	"yuoj-go-backend/internal/model/entity"
)

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

func questionEditableUpdates(q *entity.Question) map[string]interface{} {
	return map[string]interface{}{
		"title":       q.Title,
		"content":     q.Content,
		"tags":        q.Tags,
		"answer":      q.Answer,
		"sampleCase":  q.SampleCase,
		"judgeCase":   q.JudgeCase,
		"judgeConfig": q.JudgeConfig,
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
