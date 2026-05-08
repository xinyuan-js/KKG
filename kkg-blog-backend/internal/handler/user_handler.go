package handler

import (
	"strconv"
	"strings"

	"awesomeProject/internal/middleware"
	"awesomeProject/internal/service"
	"awesomeProject/pkg/response"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	users  *service.UserService
	audits *service.AdminAuditService
}

func NewUserHandler(users *service.UserService, audits *service.AdminAuditService) *UserHandler {
	return &UserHandler{users: users, audits: audits}
}

func (h *UserHandler) GetPublicPage(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || userID == 0 {
		response.BadRequest(c, "invalid user id")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	data, err := h.users.GetPublicPage(userID, limit)
	if err != nil {
		switch err.Error() {
		case "user not found":
			response.BadRequest(c, err.Error())
		case "invalid user id":
			response.BadRequest(c, err.Error())
		default:
			response.ServerError(c, err.Error())
		}
		return
	}
	response.OK(c, data)
}

func (h *UserHandler) AdminList(c *gin.Context) {
	if !middleware.IsAdminRole(middleware.MustRole(c)) {
		response.Forbidden(c, "forbidden")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	role := strings.TrimSpace(c.DefaultQuery("role", ""))
	status := strings.TrimSpace(c.DefaultQuery("status", ""))
	keyword := strings.TrimSpace(c.DefaultQuery("q", ""))
	items, total, err := h.users.ListForAdmin(keyword, role, status, page, pageSize)
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

func (h *UserHandler) AdminUpdateRole(c *gin.Context) {
	role := middleware.MustRole(c)
	actorID := middleware.MustUserID(c)
	if !middleware.IsAdminRole(role) {
		response.Forbidden(c, "forbidden")
		return
	}
	var req struct {
		ID     uint64 `json:"id"`
		Role   string `json:"role"`
		Status int8   `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid payload")
		return
	}
	if err := h.users.UpdateRoleStatusByAdmin(role, actorID, req.ID, req.Role, req.Status); err != nil {
		switch err.Error() {
		case "invalid user id", "invalid role", "invalid status", "user not found", "cannot update self":
			response.BadRequest(c, err.Error())
		case "forbidden":
			response.Forbidden(c, err.Error())
		default:
			response.ServerError(c, err.Error())
		}
		return
	}
	if h.audits != nil {
		_ = h.audits.Create(actorID, role, "user_role_or_status_update", "user", req.ID, "update role/status")
	}
	response.OK(c, gin.H{"updated": true})
}

func (h *UserHandler) AdminDelete(c *gin.Context) {
	role := middleware.MustRole(c)
	actorID := middleware.MustUserID(c)
	if role != "super_admin" {
		response.Forbidden(c, "forbidden")
		return
	}
	targetID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || targetID == 0 {
		response.BadRequest(c, "invalid user id")
		return
	}
	if err := h.users.DeleteByAdmin(role, actorID, targetID); err != nil {
		switch err.Error() {
		case "invalid user id", "cannot delete self", "user not found":
			response.BadRequest(c, err.Error())
		case "forbidden":
			response.Forbidden(c, err.Error())
		default:
			response.ServerError(c, err.Error())
		}
		return
	}
	if h.audits != nil {
		_ = h.audits.Create(actorID, role, "user_delete", "user", targetID, "delete user")
	}
	response.OK(c, gin.H{"deleted": true})
}
