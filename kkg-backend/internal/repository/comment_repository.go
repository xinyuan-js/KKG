package repository

import (
	"errors"

	"kkg-backend/internal/model"

	"gorm.io/gorm"
)

type CommentRepository struct {
	db *gorm.DB
}

const commentAuthorSelect = "CASE WHEN users.status = -1 THEN '用户已删除或注销' ELSE users.username END AS author_name, CASE WHEN users.status = -1 THEN '' ELSE users.avatar_url END AS author_avatar_url"

func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

func (r *CommentRepository) Create(comment *model.Comment) error {
	return r.db.Create(comment).Error
}

func (r *CommentRepository) GetByID(id uint64) (*model.Comment, error) {
	var c model.Comment
	err := r.db.First(&c, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CommentRepository) ListByPostID(postID uint64, limit int) ([]model.Comment, error) {
	var comments []model.Comment
	err := r.db.
		Table("comments").
		Select("comments.*, "+commentAuthorSelect).
		Joins("LEFT JOIN users ON users.id = comments.user_id").
		Where("comments.post_id = ? AND comments.status = ?", postID, "normal").
		Order("comments.created_at ASC").
		Limit(limit).
		Find(&comments).Error
	return comments, err
}
