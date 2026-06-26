package repository

import (
	"kkg-backend/internal/model"
	"strings"
)

func (r *PostRepository) SearchPublishedByKeyword(keyword string, limit int) ([]model.Post, error) {
	var posts []model.Post
	kw := "%" + keyword + "%"
	err := r.db.
		Table("posts").
		Select("posts.*, "+postAuthorSelect).
		Joins("LEFT JOIN users ON users.id = posts.author_id").
		Where("posts.status = ? AND (posts.title LIKE ? OR posts.summary LIKE ?)", "published", kw, kw).
		Order("posts.publish_at DESC").
		Limit(limit).
		Find(&posts).Error
	return posts, err
}

func (r *PostRepository) SuggestPublishedTitles(keyword string, limit int) ([]string, error) {
	var titles []string
	kw := "%" + keyword + "%"
	err := r.db.Model(&model.Post{}).
		Where("status = ? AND title LIKE ?", "published", kw).
		Order("publish_at DESC").
		Limit(limit).
		Pluck("title", &titles).Error
	return titles, err
}

func (r *PostRepository) ListForAdmin(keyword string, status string, page int, pageSize int) ([]model.Post, int64, error) {
	q := r.db.Model(&model.Post{})
	kw := strings.TrimSpace(keyword)
	if kw != "" {
		like := "%" + kw + "%"
		q = q.Where("title LIKE ? OR summary LIKE ?", like, like)
	}
	sv := strings.TrimSpace(status)
	if sv != "" {
		q = q.Where("status = ?", sv)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var posts []model.Post
	err := r.db.
		Table("posts").
		Select("posts.*, "+postAuthorSelect).
		Joins("LEFT JOIN users ON users.id = posts.author_id").
		Where("posts.id IN (?)", q.Select("id")).
		Order("posts.updated_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&posts).Error
	return posts, total, err
}
