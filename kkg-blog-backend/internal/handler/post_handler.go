package handler

import (
	"strconv"
	"strings"

	"awesomeProject/internal/middleware"
	"awesomeProject/internal/service"
	"awesomeProject/pkg/response"

	"github.com/gin-gonic/gin"
)

type PostHandler struct {
	posts  *service.PostService
	audits *service.AdminAuditService
}

func NewPostHandler(posts *service.PostService, audits *service.AdminAuditService) *PostHandler {
	return &PostHandler{posts: posts, audits: audits}
}

type createPostReq struct {
	Title      string   `json:"title" binding:"required,max=255"`
	Slug       string   `json:"slug" binding:"max=255"`
	Summary    string   `json:"summary" binding:"max=512"`
	Tags       []string `json:"tags" binding:"omitempty,dive,max=32"`
	DraftNote  *string  `json:"draft_note" binding:"omitempty,max=1024"`
	RawContent string   `json:"raw_content"`
}

type saveDraftReq struct {
	Title      string  `json:"title" binding:"max=255"`
	Summary    string  `json:"summary" binding:"max=512"`
	DraftNote  *string `json:"draft_note" binding:"omitempty,max=1024"`
	RawContent string  `json:"raw_content"`
}

type createDraftCopyReq struct {
	FromVersion int     `json:"from_version"`
	DraftNote   *string `json:"draft_note" binding:"omitempty,max=1024"`
}

type updatePostMetaReq struct {
	Title   string   `json:"title" binding:"required,max=255"`
	Summary string   `json:"summary" binding:"max=512"`
	Tags    []string `json:"tags" binding:"omitempty,dive,max=32"`
}

func (h *PostHandler) CreateDraft(c *gin.Context) {
	var req createPostReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	userID := middleware.MustUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "invalid user context")
		return
	}

	post, err := h.posts.CreateDraft(userID, req.Title, req.Slug, req.Summary, req.Tags, req.RawContent, req.DraftNote)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, post)
}

func (h *PostHandler) UpdateMeta(c *gin.Context) {
	postID, err := parseID(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}
	userID := middleware.MustUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "invalid user context")
		return
	}
	var req updatePostMetaReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	post, err := h.posts.UpdateMeta(postID, userID, req.Title, req.Summary, req.Tags)
	if err != nil {
		h.handlePostErr(c, err)
		return
	}
	response.OK(c, post)
}

func (h *PostHandler) CreateDraftCopy(c *gin.Context) {
	postID, err := parseID(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}
	userID := middleware.MustUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "invalid user context")
		return
	}

	var req createDraftCopyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		// allow empty body
		req.FromVersion = 0
	}

	draft, err := h.posts.CreateDraftCopy(postID, userID, req.FromVersion, req.DraftNote)
	if err != nil {
		h.handlePostErr(c, err)
		return
	}
	response.OK(c, draft)
}

func (h *PostHandler) SaveDraft(c *gin.Context) {
	postID, err := parseID(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}

	var req saveDraftReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID := middleware.MustUserID(c)
	post, err := h.posts.SaveDraft(postID, userID, req.Title, req.Summary, req.RawContent)
	if err != nil {
		h.handlePostErr(c, err)
		return
	}
	response.OK(c, post)
}

func (h *PostHandler) SaveDraftByVersion(c *gin.Context) {
	postID, err := parseID(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}
	version, err := strconv.Atoi(c.Param("version"))
	if err != nil || version <= 0 {
		response.BadRequest(c, "invalid version")
		return
	}
	var req saveDraftReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	userID := middleware.MustUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "invalid user context")
		return
	}
	draft, err := h.posts.SaveDraftByVersion(postID, userID, version, req.Title, req.Summary, req.RawContent, req.DraftNote)
	if err != nil {
		h.handlePostErr(c, err)
		return
	}
	response.OK(c, draft)
}

func (h *PostHandler) Publish(c *gin.Context) {
	postID, err := parseID(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}
	userID := middleware.MustUserID(c)

	post, err := h.posts.Publish(postID, userID)
	if err != nil {
		h.handlePostErr(c, err)
		return
	}
	response.OK(c, post)
}

func (h *PostHandler) Unpublish(c *gin.Context) {
	postID, err := parseID(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}
	userID := middleware.MustUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "invalid user context")
		return
	}
	post, err := h.posts.Unpublish(postID, userID)
	if err != nil {
		h.handlePostErr(c, err)
		return
	}
	response.OK(c, post)
}

func (h *PostHandler) PublishDraft(c *gin.Context) {
	postID, err := parseID(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}
	version, err := strconv.Atoi(c.Param("version"))
	if err != nil || version <= 0 {
		response.BadRequest(c, "invalid version")
		return
	}
	userID := middleware.MustUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "invalid user context")
		return
	}
	post, err := h.posts.PublishDraft(postID, userID, version)
	if err != nil {
		h.handlePostErr(c, err)
		return
	}
	response.OK(c, post)
}

func (h *PostHandler) Get(c *gin.Context) {
	postID, err := parseID(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}

	actorID := middleware.MustUserID(c)
	post, err := h.posts.Get(postID, actorID)
	if err != nil {
		h.handlePostErr(c, err)
		return
	}
	response.OK(c, post)
}

func (h *PostHandler) GetMine(c *gin.Context) {
	postID, err := parseID(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}
	userID := middleware.MustUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "invalid user context")
		return
	}
	post, err := h.posts.GetMine(postID, userID)
	if err != nil {
		h.handlePostErr(c, err)
		return
	}
	response.OK(c, post)
}

func (h *PostHandler) GetDraft(c *gin.Context) {
	postID, err := parseID(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}
	version, err := strconv.Atoi(c.Param("version"))
	if err != nil || version <= 0 {
		response.BadRequest(c, "invalid version")
		return
	}
	userID := middleware.MustUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "invalid user context")
		return
	}
	draft, err := h.posts.GetDraft(postID, userID, version)
	if err != nil {
		h.handlePostErr(c, err)
		return
	}
	response.OK(c, draft)
}

func (h *PostHandler) ListPublished(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	posts, err := h.posts.ListPublished(limit)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, posts)
}

func (h *PostHandler) ListFeed(c *gin.Context) {
	feedType := c.DefaultQuery("type", "latest")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	actorID := middleware.MustUserID(c)
	posts, err := h.posts.ListFeed(feedType, actorID, limit)
	if err != nil {
		if err.Error() == "invalid feed type" {
			response.BadRequest(c, err.Error())
			return
		}
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, gin.H{
		"type":  feedType,
		"items": posts,
	})
}

func (h *PostHandler) ToggleLike(c *gin.Context) {
	postID, err := parseID(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}
	userID := middleware.MustUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "invalid user context")
		return
	}
	eng, err := h.posts.ToggleLike(postID, userID)
	if err != nil {
		h.handlePostErr(c, err)
		return
	}
	response.OK(c, eng)
}

func (h *PostHandler) ToggleFavorite(c *gin.Context) {
	postID, err := parseID(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}
	userID := middleware.MustUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "invalid user context")
		return
	}
	eng, err := h.posts.ToggleFavorite(postID, userID)
	if err != nil {
		h.handlePostErr(c, err)
		return
	}
	response.OK(c, eng)
}

func (h *PostHandler) GetEngagement(c *gin.Context) {
	postID, err := parseID(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}
	userID := middleware.MustUserID(c)
	eng, err := h.posts.GetEngagement(postID, userID)
	if err != nil {
		h.handlePostErr(c, err)
		return
	}
	response.OK(c, eng)
}

func (h *PostHandler) Ranking(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	period := c.DefaultQuery("period", "all")
	posts, err := h.posts.ListRankingByPeriod(limit, period)
	if err != nil {
		if err.Error() == "invalid ranking period" {
			response.BadRequest(c, err.Error())
			return
		}
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, gin.H{
		"period": period,
		"items":  posts,
	})
}

func (h *PostHandler) ListMyFavorites(c *gin.Context) {
	userID := middleware.MustUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "invalid user context")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	posts, err := h.posts.ListMyFavorites(userID, limit)
	if err != nil {
		h.handlePostErr(c, err)
		return
	}
	response.OK(c, posts)
}

func (h *PostHandler) ListMine(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	userID := middleware.MustUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "invalid user context")
		return
	}

	posts, err := h.posts.ListMine(userID, limit)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, posts)
}

func (h *PostHandler) AdminList(c *gin.Context) {
	if !middleware.IsAdminRole(middleware.MustRole(c)) {
		response.Forbidden(c, "forbidden")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	keyword := strings.TrimSpace(c.DefaultQuery("q", ""))
	status := strings.TrimSpace(c.DefaultQuery("status", ""))
	items, total, err := h.posts.ListForAdmin(keyword, status, page, pageSize)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *PostHandler) ListVersions(c *gin.Context) {
	postID, err := parseID(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	userID := middleware.MustUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "invalid user context")
		return
	}
	versions, err := h.posts.ListVersions(postID, userID, limit)
	if err != nil {
		h.handlePostErr(c, err)
		return
	}
	response.OK(c, versions)
}

func (h *PostHandler) ListDrafts(c *gin.Context) {
	postID, err := parseID(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	userID := middleware.MustUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "invalid user context")
		return
	}
	drafts, err := h.posts.ListDrafts(postID, userID, limit)
	if err != nil {
		h.handlePostErr(c, err)
		return
	}
	response.OK(c, drafts)
}

func (h *PostHandler) Rollback(c *gin.Context) {
	postID, err := parseID(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}
	version, err := strconv.Atoi(c.Param("version"))
	if err != nil || version <= 0 {
		response.BadRequest(c, "invalid version")
		return
	}
	userID := middleware.MustUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "invalid user context")
		return
	}
	post, err := h.posts.Rollback(postID, userID, version)
	if err != nil {
		h.handlePostErr(c, err)
		return
	}
	response.OK(c, post)
}

func (h *PostHandler) DeleteVersion(c *gin.Context) {
	postID, err := parseID(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}
	version, err := strconv.Atoi(c.Param("version"))
	if err != nil || version <= 0 {
		response.BadRequest(c, "invalid version")
		return
	}
	userID := middleware.MustUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "invalid user context")
		return
	}
	if err := h.posts.DeleteVersion(postID, userID, version); err != nil {
		h.handlePostErr(c, err)
		return
	}
	response.OK(c, gin.H{"deleted_version": version})
}

func (h *PostHandler) DeleteDraft(c *gin.Context) {
	postID, err := parseID(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}
	version, err := strconv.Atoi(c.Param("version"))
	if err != nil || version <= 0 {
		response.BadRequest(c, "invalid version")
		return
	}
	userID := middleware.MustUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "invalid user context")
		return
	}
	if err := h.posts.DeleteDraft(postID, userID, version); err != nil {
		h.handlePostErr(c, err)
		return
	}
	response.OK(c, gin.H{"deleted_version": version})
}

func (h *PostHandler) DeletePost(c *gin.Context) {
	postID, err := parseID(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}
	userID := middleware.MustUserID(c)
	role := middleware.MustRole(c)
	if userID == 0 {
		response.Unauthorized(c, "invalid user context")
		return
	}
	var errDelete error
	if middleware.IsAdminRole(role) {
		errDelete = h.posts.DeletePostAsManager(postID)
	} else {
		errDelete = h.posts.DeletePost(postID, userID)
	}
	if errDelete != nil {
		h.handlePostErr(c, errDelete)
		return
	}
	if middleware.IsAdminRole(role) && h.audits != nil {
		_ = h.audits.Create(userID, role, "post_hide", "post", postID, "hide post")
	}
	response.OK(c, gin.H{"deleted_post_id": postID})
}

func (h *PostHandler) handlePostErr(c *gin.Context, err error) {
	if err == nil {
		return
	}
	switch err.Error() {
	case "post not found":
		response.BadRequest(c, err.Error())
	case "version not found":
		response.BadRequest(c, err.Error())
	case "invalid version":
		response.BadRequest(c, err.Error())
	case "already latest version":
		response.BadRequest(c, err.Error())
	case "cannot delete current version":
		response.BadRequest(c, err.Error())
	case "at least one version must be kept":
		response.BadRequest(c, err.Error())
	case "cannot delete published draft":
		response.BadRequest(c, err.Error())
	case "at least one draft must be kept":
		response.BadRequest(c, err.Error())
	case "cannot edit published draft":
		response.BadRequest(c, err.Error())
	case "post is not published":
		response.BadRequest(c, err.Error())
	case "title is required":
		response.BadRequest(c, err.Error())
	case "forbidden":
		response.Forbidden(c, err.Error())
	default:
		response.ServerError(c, err.Error())
	}
}

func parseID(raw string) (uint64, error) {
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	return id, nil
}
