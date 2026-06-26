package handler

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"kkg-backend/internal/oj/common"
	"kkg-backend/internal/oj/model/entity"
)

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
	mustNoErr(h.db.Where("id=?", req.ID).First(&q).Error)
	if q.UserID != login.ID && !h.userSvc.IsAdmin(login) {
		panic(common.NewBizError(common.NoAuthError, "无权限"))
	}
	mustNoErr(h.db.Model(&entity.Question{}).Where("id=?", req.ID).Update("isDelete", 1).Error)
	c.JSON(http.StatusOK, common.Success(true))
}
func (h *Handler) QuestionRestore(c *gin.Context) {
	h.mustAdmin(c)
	var req common.DeleteRequest
	mustBindJSON(c, &req)
	if req.ID <= 0 {
		panic(common.NewBizError(common.ParamsError, "请求参数错误"))
	}
	mustNoErr(h.db.Model(&entity.Question{}).Where("id=?", req.ID).Update("isDelete", 0).Error)
	c.JSON(http.StatusOK, common.Success(true))
}
func (h *Handler) QuestionUpdate(c *gin.Context) {
	h.mustAdmin(c)
	var q entity.Question
	mustBindJSON(c, &q)
	validQuestionInput(&q, false)
	mustNoErr(h.db.Model(&entity.Question{}).Where("id=? AND isDelete=0", q.ID).Updates(questionEditableUpdates(&q)).Error)
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
	mustNoErr(h.db.Model(&entity.Question{}).Where("id=? AND isDelete=0", q.ID).Updates(questionEditableUpdates(&q)).Error)
	c.JSON(http.StatusOK, common.Success(true))
}

func (h *Handler) questionList(c *gin.Context, onlyMine bool, raw bool, mine ...int64) {
	var req struct {
		common.PageRequest
		UserID     int64    `json:"userId"`
		Title      string   `json:"title"`
		SearchText string   `json:"searchText"`
		Tags       []string `json:"tags"`
		Status     string   `json:"status"`
	}
	mustBindJSON(c, &req)
	req.Normalize()
	if req.PageSize > 20 && !raw {
		panic(common.NewBizError(common.ParamsError, "请求参数错误"))
	}
	var list []entity.Question
	query := h.db.Model(&entity.Question{})
	if raw {
		switch strings.TrimSpace(req.Status) {
		case "hidden":
			query = query.Where("isDelete=1")
		case "all", "":
		default:
			query = query.Where("isDelete=0")
		}
	} else {
		query = query.Where("isDelete=0")
	}
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

func questionVO(q *entity.Question) map[string]interface{} {
	var tags []string
	var cfg map[string]interface{}
	var sampleCases []judgeCase
	_ = json.Unmarshal([]byte(q.Tags), &tags)
	_ = json.Unmarshal([]byte(q.JudgeConfig), &cfg)
	_ = json.Unmarshal([]byte(q.SampleCase), &sampleCases)
	return map[string]interface{}{"id": q.ID, "title": q.Title, "content": q.Content, "tags": tags, "submitNum": q.SubmitNum, "acceptedNum": q.AcceptedNum, "judgeConfig": cfg, "sampleCase": sampleCases, "thumbNum": q.ThumbNum, "favourNum": q.FavourNum, "userId": q.UserID, "createTime": q.CreateTime, "updateTime": q.UpdateTime}
}
