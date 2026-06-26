package bootstrap

import (
	"fmt"

	"kkg-backend/internal/model"
	ojentity "kkg-backend/internal/oj/model/entity"

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
		&model.UserIdentityMap{},
		&ojentity.User{},
		&ojentity.Question{},
		&ojentity.QuestionSubmit{},
		&ojentity.QuestionSolutionPost{},
		&ojentity.AgentSolutionTask{},
	); err != nil {
		return fmt.Errorf("auto migrate failed: %w", err)
	}
	return nil
}
