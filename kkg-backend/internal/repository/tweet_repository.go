package repository

import (
	"kkg-backend/internal/model"

	"gorm.io/gorm"
)

type TweetRepository struct {
	db *gorm.DB
}

func NewTweetRepository(db *gorm.DB) *TweetRepository {
	return &TweetRepository{db: db}
}

func (r *TweetRepository) DB() *gorm.DB {
	return r.db
}

func (r *TweetRepository) Create(tweet *model.Tweet) error {
	return r.db.Create(tweet).Error
}
