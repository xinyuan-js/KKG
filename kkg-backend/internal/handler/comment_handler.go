package handler

import (
	"strconv"

	"kkg-backend/internal/middleware"
	"kkg-backend/internal/service"
	"kkg-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type CommentHandler struct {
	comments *service.CommentService
}

func NewCommentHandler(comments *service.CommentService) *CommentHandler {
	return &CommentHandler{comments: comments}
}

type createCommentReq struct {
	ParentID *uint64 `json:"parent_id"`
	Content  string  `json:"content" binding:"required"`
}

func (h *CommentHandler) ListByPost(c *gin.Context) {
	postID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || postID == 0 {
		response.BadRequest(c, "invalid post id")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "200"))
	data, err := h.comments.ListByPost(postID, limit)
	if err != nil {
		switch err.Error() {
		case "invalid post id", "post not found":
			response.BadRequest(c, err.Error())
		case "forbidden":
			response.Forbidden(c, err.Error())
		default:
			response.ServerError(c, err.Error())
		}
		return
	}
	response.OK(c, data)
}

func (h *CommentHandler) CreateByPost(c *gin.Context) {
	postID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || postID == 0 {
		response.BadRequest(c, "invalid post id")
		return
	}
	userID := middleware.MustUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "invalid user context")
		return
	}
	var req createCommentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	data, err := h.comments.Create(postID, userID, req.ParentID, req.Content)
	if err != nil {
		switch err.Error() {
		case "invalid post id", "content is required", "content too long", "post not found", "parent comment not found":
			response.BadRequest(c, err.Error())
		case "cannot comment unpublished post":
			response.Forbidden(c, err.Error())
		default:
			response.ServerError(c, err.Error())
		}
		return
	}
	response.OK(c, data)
}
