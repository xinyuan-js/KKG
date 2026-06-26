package handler

import (
	"awesomeProject/internal/service"
	"awesomeProject/pkg/response"
	"github.com/gin-gonic/gin"
	"strconv"
)

type PostHandler struct {
	posts  *service.PostService
	audits *service.AdminAuditService
}

func NewPostHandler(posts *service.PostService, audits *service.AdminAuditService) *PostHandler {
	return &PostHandler{posts: posts, audits: audits}
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
