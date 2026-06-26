package handler

import (
	"awesomeProject/internal/middleware"
	"awesomeProject/pkg/response"
	"github.com/gin-gonic/gin"
	"strconv"
)

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
