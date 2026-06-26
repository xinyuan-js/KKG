package service

import (
	"errors"
	"strings"

	"kkg-backend/internal/middleware"
	"kkg-backend/internal/model"
	"kkg-backend/internal/repository"
)

type AdminAuditService struct {
	audits *repository.AdminAuditRepository
}

func NewAdminAuditService(audits *repository.AdminAuditRepository) *AdminAuditService {
	return &AdminAuditService{audits: audits}
}

func (s *AdminAuditService) Create(actorID uint64, actorRole string, action string, targetType string, targetID uint64, detail string) error {
	if actorID == 0 || targetID == 0 {
		return errors.New("invalid actor or target")
	}
	actorRole = strings.TrimSpace(actorRole)
	if !middleware.IsAdminRole(actorRole) {
		return errors.New("forbidden")
	}
	action = strings.TrimSpace(action)
	targetType = strings.TrimSpace(targetType)
	if action == "" || targetType == "" {
		return errors.New("invalid action or target")
	}
	log := &model.AdminAuditLog{
		ActorID:    actorID,
		ActorRole:  actorRole,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Detail:     strings.TrimSpace(detail),
	}
	return s.audits.Create(log)
}

func (s *AdminAuditService) List(page int, pageSize int, action string) ([]model.AdminAuditLog, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	return s.audits.List(page, pageSize, action)
}
