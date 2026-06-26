package bootstrap

import (
	"kkg-backend/internal/config"
	"kkg-backend/internal/model"
	"kkg-backend/pkg/security"
	"errors"
	"gorm.io/gorm"
	"strings"
)

func ensureSuperAdmin(db *gorm.DB, cfg *config.Config) error {
	username := strings.TrimSpace(cfg.SuperAdminUsername)
	email := strings.TrimSpace(cfg.SuperAdminEmail)
	password := strings.TrimSpace(cfg.SuperAdminPassword)
	if username == "" || email == "" || password == "" {
		return nil
	}
	if len(password) < 8 {
		return errors.New("SUPER_ADMIN_PASSWORD must be at least 8 characters")
	}

	var user model.User
	err := db.Where("username = ? OR email = ?", username, email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		hash, hashErr := security.HashPassword(password)
		if hashErr != nil {
			return hashErr
		}
		u := &model.User{
			Username:     username,
			Email:        email,
			PasswordHash: hash,
			Role:         "super_admin",
			Status:       1,
		}
		return db.Create(u).Error
	}
	if err != nil {
		return err
	}

	updates := map[string]interface{}{}
	if user.Role != "super_admin" {
		updates["role"] = "super_admin"
	}
	if user.Status != 1 {
		updates["status"] = 1
	}
	if len(updates) == 0 {
		return nil
	}
	return db.Model(&model.User{}).Where("id = ?", user.ID).Updates(updates).Error
}
