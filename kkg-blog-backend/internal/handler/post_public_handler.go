package handler

import (
	"awesomeProject/internal/middleware"
	"awesomeProject/pkg/response"
	"github.com/gin-gonic/gin"
	"strconv"
)

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
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", c.DefaultQuery("limit", "20")))
	actorID := middleware.MustUserID(c)
	posts, total, err := h.posts.PageFeed(feedType, actorID, page, pageSize)
	if err != nil {
		if err.Error() == "invalid feed type" {
			response.BadRequest(c, err.Error())
			return
		}
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, gin.H{
		"type":      feedType,
		"items":     posts,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
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
