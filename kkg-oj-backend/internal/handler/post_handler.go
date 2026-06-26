package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"time"
	"yuoj-go-backend/internal/common"
	searchintegration "yuoj-go-backend/internal/integration/search"
	"yuoj-go-backend/internal/model/entity"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

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

func (h *Handler) searchPostIDsFromES(current, pageSize, userID int64, title, content, searchText string, tags []string) ([]int64, int64, error) {
	return searchintegration.NewElasticsearchClient(h.cfg.ES.URL, h.cfg.ES.Index, 5*time.Second).SearchPostIDs(searchintegration.PostQuery{
		Current:    current,
		PageSize:   pageSize,
		UserID:     userID,
		Title:      title,
		Content:    content,
		SearchText: searchText,
		Tags:       tags,
	})
}
