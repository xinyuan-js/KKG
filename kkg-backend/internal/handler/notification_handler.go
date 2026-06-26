package handler

import (
	"strconv"

	"kkg-backend/internal/middleware"
	"kkg-backend/internal/service"
	"kkg-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	notifications *service.NotificationService
}

func NewNotificationHandler(notifications *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notifications: notifications}
}

func (h *NotificationHandler) ListMine(c *gin.Context) {
	userID := middleware.MustUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "invalid user context")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	data, err := h.notifications.ListMine(userID, limit)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, data)
}

func (h *NotificationHandler) MarkRead(c *gin.Context) {
	userID := middleware.MustUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "invalid user context")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.BadRequest(c, "invalid notification id")
		return
	}
	if err := h.notifications.MarkRead(userID, id); err != nil {
		switch err.Error() {
		case "invalid request", "notification not found":
			response.BadRequest(c, err.Error())
		case "forbidden":
			response.Forbidden(c, err.Error())
		default:
			response.ServerError(c, err.Error())
		}
		return
	}
	response.OK(c, gin.H{"read": true, "id": id})
}
