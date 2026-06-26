package handler

import (
	"strconv"

	"kkg-backend/internal/middleware"
	"kkg-backend/internal/service"
	"kkg-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type TweetHandler struct {
	tweets *service.TweetService
}

func NewTweetHandler(tweets *service.TweetService) *TweetHandler {
	return &TweetHandler{tweets: tweets}
}

type createTweetReq struct {
	Content string `json:"content" binding:"required"`
}

func (h *TweetHandler) Create(c *gin.Context) {
	userID := middleware.MustUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "invalid user context")
		return
	}
	var req createTweetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	tweet, err := h.tweets.Create(userID, req.Content)
	if err != nil {
		switch err.Error() {
		case "invalid user context", "content is required", "content too long, max 280 characters":
			response.BadRequest(c, err.Error())
		default:
			response.ServerError(c, err.Error())
		}
		return
	}
	response.OK(c, tweet)
}

func (h *TweetHandler) Search(c *gin.Context) {
	q := c.Query("q")
	from, _ := strconv.Atoi(c.DefaultQuery("from", "0"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	items, err := h.tweets.Search(q, from, size)
	if err != nil {
		if err.Error() == "query is required" {
			response.BadRequest(c, err.Error())
			return
		}
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, gin.H{
		"items": items,
		"from":  from,
		"size":  size,
	})
}
