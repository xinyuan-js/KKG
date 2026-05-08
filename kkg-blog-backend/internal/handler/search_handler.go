package handler

import (
	"strconv"
	"strings"

	"awesomeProject/internal/middleware"
	"awesomeProject/internal/service"
	"awesomeProject/pkg/response"

	"github.com/gin-gonic/gin"
)

type SearchHandler struct {
	search *service.SearchService
	users  *service.UserService
}

func NewSearchHandler(search *service.SearchService, users *service.UserService) *SearchHandler {
	return &SearchHandler{search: search, users: users}
}

func (h *SearchHandler) Search(c *gin.Context) {
	typ := strings.TrimSpace(c.DefaultQuery("type", "post"))
	q := c.Query("q")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	switch typ {
	case "post":
		posts, err := h.search.SearchPosts(q, limit)
		if err != nil {
			if err.Error() == "query is required" {
				response.BadRequest(c, err.Error())
				return
			}
			response.ServerError(c, err.Error())
			return
		}
		response.OK(c, gin.H{"type": "post", "items": posts})
	case "user":
		excludeID := middleware.MustUserID(c)
		users, err := h.users.SearchPublicUsers(q, excludeID, limit)
		if err != nil {
			if err.Error() == "query is required" {
				response.BadRequest(c, err.Error())
				return
			}
			response.ServerError(c, err.Error())
			return
		}
		response.OK(c, gin.H{"type": "user", "items": users})
	default:
		response.BadRequest(c, "invalid search type")
	}
}

func (h *SearchHandler) Suggest(c *gin.Context) {
	typ := strings.TrimSpace(c.DefaultQuery("type", "post"))
	q := c.Query("q")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "5"))

	switch typ {
	case "post":
		items, err := h.search.SuggestPosts(q, limit)
		if err != nil {
			response.ServerError(c, err.Error())
			return
		}
		response.OK(c, gin.H{"type": "post", "items": items})
	case "user":
		excludeID := middleware.MustUserID(c)
		items, err := h.users.SuggestPublicUsernames(q, excludeID, limit)
		if err != nil {
			response.ServerError(c, err.Error())
			return
		}
		response.OK(c, gin.H{"type": "user", "items": items})
	default:
		response.BadRequest(c, "invalid search type")
	}
}
