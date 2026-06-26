package repository

import (
	"kkg-backend/internal/model"
	"gorm.io/gorm"
)

type PostRepository struct {
	db *gorm.DB
}

const postAuthorSelect = "CASE WHEN users.status = -1 THEN '用户已删除或注销' ELSE users.username END AS author_name, CASE WHEN users.status = -1 THEN '' ELSE users.avatar_url END AS author_avatar_url"

func NewPostRepository(db *gorm.DB) *PostRepository {
	return &PostRepository{db: db}
}

func (r *PostRepository) DB() *gorm.DB {
	return r.db
}

func (r *PostRepository) Create(post *model.Post) error {
	return r.db.Create(post).Error
}

func (r *PostRepository) ExistsSlug(slug string) (bool, error) {
	var count int64
	err := r.db.Unscoped().Model(&model.Post{}).Where("slug = ?", slug).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *PostRepository) ExistsSlugForAuthor(authorID uint64, slug string) (bool, error) {
	var count int64
	err := r.db.Unscoped().Model(&model.Post{}).Where("author_id = ? AND slug = ?", authorID, slug).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *PostRepository) Update(post *model.Post) error {
	return r.db.Save(post).Error
}
