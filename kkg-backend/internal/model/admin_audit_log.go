package model

import "time"

type AdminAuditLog struct {
	ID         uint64    `gorm:"primaryKey" json:"id"`
	ActorID    uint64    `gorm:"not null;index" json:"actor_id"`
	ActorRole  string    `gorm:"size:32;not null;index" json:"actor_role"`
	Action     string    `gorm:"size:64;not null;index" json:"action"`
	TargetType string    `gorm:"size:32;not null;index" json:"target_type"`
	TargetID   uint64    `gorm:"not null;index" json:"target_id"`
	Detail     string    `gorm:"type:text" json:"detail"`
	CreatedAt  time.Time `json:"created_at"`

	ActorName string `gorm:"->;column:actor_name" json:"actor_name,omitempty"`
}

func (AdminAuditLog) TableName() string {
	return "admin_audit_logs"
}
