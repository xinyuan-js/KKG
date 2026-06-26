package model

import "time"

type UserIdentityMap struct {
	ID              uint64    `gorm:"primaryKey" json:"id"`
	AuthUserID      uint64    `gorm:"not null;uniqueIndex:uk_user_identity_auth_source" json:"auth_user_id"`
	LegacyOJUserID  *uint64   `gorm:"uniqueIndex:uk_user_identity_oj_user" json:"legacy_oj_user_id"`
	LegacyOJAccount string    `gorm:"size:128;not null;default:'';index:idx_user_identity_oj_account" json:"legacy_oj_account"`
	Source          string    `gorm:"size:32;not null;default:oj;uniqueIndex:uk_user_identity_auth_source" json:"source"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (UserIdentityMap) TableName() string {
	return "user_identity_map"
}
