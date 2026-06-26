package bootstrap

import (
	"fmt"

	"awesomeProject/internal/model"

	"gorm.io/gorm"
)

func migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&model.User{},
		&model.Post{},
		&model.PostVersion{},
		&model.Comment{},
		&model.Notification{},
		&model.Tweet{},
		&model.PostLike{},
		&model.PostFavorite{},
		&model.AdminAuditLog{},
	); err != nil {
		return fmt.Errorf("auto migrate failed: %w", err)
	}
	return nil
}
