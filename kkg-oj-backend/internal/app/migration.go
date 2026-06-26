package app

import (
	"yuoj-go-backend/internal/model/entity"

	"gorm.io/gorm"
)

func migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&entity.User{},
		&entity.Question{},
		&entity.QuestionSubmit{},
		&entity.QuestionSolutionPost{},
		&entity.AgentSolutionTask{},
	)
}
