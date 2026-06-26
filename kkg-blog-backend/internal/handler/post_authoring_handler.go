package handler

import (
	"awesomeProject/internal/middleware"
	"awesomeProject/pkg/response"
	"github.com/gin-gonic/gin"
	"strconv"
)

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
