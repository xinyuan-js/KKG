package handler

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"time"
	"yuoj-go-backend/internal/common"
	blogintegration "yuoj-go-backend/internal/integration/blog"
	"yuoj-go-backend/internal/model/entity"
)

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
	return blogintegration.NewClient(h.cfg.Blog.BaseURL, h.cfg.Blog.InternalAuthToken, 2*time.Second).FetchPostPreviewMap(postIDs)
}
