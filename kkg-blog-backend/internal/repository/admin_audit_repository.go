package repository

import (
	"awesomeProject/internal/model"
	"strings"

	"gorm.io/gorm"
)

type AdminAuditRepository struct {
	db *gorm.DB
}

func NewAdminAuditRepository(db *gorm.DB) *AdminAuditRepository {
	return &AdminAuditRepository{db: db}
}

func (r *AdminAuditRepository) Create(log *model.AdminAuditLog) error {
	return r.db.Create(log).Error
}

func (r *AdminAuditRepository) List(page int, pageSize int, action string) ([]model.AdminAuditLog, int64, error) {
	q := r.db.Model(&model.AdminAuditLog{})
	action = strings.TrimSpace(action)
	if action != "" {
		q = q.Where("action = ?", action)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []model.AdminAuditLog
	err := r.db.
		Table("admin_audit_logs").
		Select("admin_audit_logs.*, users.username AS actor_name").
		Joins("LEFT JOIN users ON users.id = admin_audit_logs.actor_id").
		Where("admin_audit_logs.id IN (?)", q.Select("id")).
		Order("admin_audit_logs.id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&rows).Error
	return rows, total, err
}

