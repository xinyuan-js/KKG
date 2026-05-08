package handler

import (
	"strconv"
	"strings"

	"awesomeProject/internal/middleware"
	"awesomeProject/internal/service"
	"awesomeProject/pkg/response"

	"github.com/gin-gonic/gin"
)

type AdminAuditHandler struct {
	audit *service.AdminAuditService
}

func NewAdminAuditHandler(audit *service.AdminAuditService) *AdminAuditHandler {
	return &AdminAuditHandler{audit: audit}
}

func (h *AdminAuditHandler) Create(c *gin.Context) {
	role := middleware.MustRole(c)
	actorID := middleware.MustUserID(c)
	if !middleware.IsAdminRole(role) {
		response.Forbidden(c, "forbidden")
		return
	}
	var req struct {
		Action     string `json:"action"`
		TargetType string `json:"target_type"`
		TargetID   uint64 `json:"target_id"`
		Detail     string `json:"detail"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid payload")
		return
	}
	if err := h.audit.Create(actorID, role, req.Action, req.TargetType, req.TargetID, req.Detail); err != nil {
		switch err.Error() {
		case "forbidden":
			response.Forbidden(c, err.Error())
		case "invalid actor or target", "invalid action or target":
			response.BadRequest(c, err.Error())
		default:
			response.ServerError(c, err.Error())
		}
		return
	}
	response.OK(c, gin.H{"created": true})
}

func (h *AdminAuditHandler) List(c *gin.Context) {
	role := middleware.MustRole(c)
	if !middleware.IsAdminRole(role) {
		response.Forbidden(c, "forbidden")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	action := strings.TrimSpace(c.DefaultQuery("action", ""))
	items, total, err := h.audit.List(page, pageSize, action)
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
