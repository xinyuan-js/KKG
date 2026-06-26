package handler

import (
	"kkg-backend/internal/middleware"
	"kkg-backend/pkg/response"
	"github.com/gin-gonic/gin"
	"strconv"
	"strings"
)

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
