package repository

import (
	"kkg-backend/internal/model"
	"errors"
	"gorm.io/gorm"
)

func (r *PostRepository) GetByID(id uint64) (*model.Post, error) {
	var post model.Post
	err := r.db.
		Table("posts").
		Select("posts.*, "+postAuthorSelect).
		Joins("LEFT JOIN users ON users.id = posts.author_id").
		Where("posts.id = ?", id).
		First(&post).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &post, nil
}

func (r *PostRepository) GetByIDForAuthor(id uint64, authorID uint64) (*model.Post, error) {
	var post model.Post
	err := r.db.Where("id = ? AND author_id = ?", id, authorID).First(&post).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &post, nil
}

func (r *PostRepository) ListPublished(limit int) ([]model.Post, error) {
	var posts []model.Post
	err := r.db.
		Table("posts").
		Select("posts.*, "+postAuthorSelect).
		Joins("LEFT JOIN users ON users.id = posts.author_id").
		Where("posts.status = ?", "published").
		Order("posts.publish_at DESC").
		Limit(limit).
		Find(&posts).Error
	return posts, err
}

func (r *PostRepository) PagePublished(page int, pageSize int) ([]model.Post, int64, error) {
	var total int64
	if err := r.db.Model(&model.Post{}).Where("status = ?", "published").Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var posts []model.Post
	err := r.db.
		Table("posts").
		Select("posts.*, "+postAuthorSelect).
		Joins("LEFT JOIN users ON users.id = posts.author_id").
		Where("posts.status = ?", "published").
		Order("posts.publish_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&posts).Error
	return posts, total, err
}

func (r *PostRepository) ListByAuthor(authorID uint64, limit int) ([]model.Post, error) {
	var posts []model.Post
	err := r.db.Where("author_id = ?", authorID).Order("updated_at DESC").Limit(limit).Find(&posts).Error
	return posts, err
}

func (r *PostRepository) ListPublishedByAuthor(authorID uint64, limit int) ([]model.Post, error) {
	var posts []model.Post
	err := r.db.
		Table("posts").
		Select("posts.*, "+postAuthorSelect).
		Joins("LEFT JOIN users ON users.id = posts.author_id").
		Where("posts.author_id = ? AND posts.status = ?", authorID, "published").
		Order("posts.publish_at DESC").
		Limit(limit).
		Find(&posts).Error
	return posts, err
}

func (r *PostRepository) CountPublishedByAuthor(authorID uint64) (int64, error) {
	var count int64
	err := r.db.Model(&model.Post{}).
		Where("author_id = ? AND status = ?", authorID, "published").
		Count(&count).Error
	return count, err
}

func (r *PostRepository) ListPublishedWithCommentCount(limit int) ([]model.Post, error) {
	var posts []model.Post
	err := r.db.
		Table("posts").
		Select("posts.*, "+postAuthorSelect+", COUNT(comments.id) AS comment_count").
		Joins("LEFT JOIN users ON users.id = posts.author_id").
		Joins("LEFT JOIN comments ON comments.post_id = posts.id AND comments.status = ?", "normal").
		Where("posts.status = ?", "published").
		Group("posts.id").
		Order("posts.publish_at DESC").
		Limit(limit).
		Find(&posts).Error
	return posts, err
}

func (r *PostRepository) ListPublishedByAuthors(authorIDs []uint64, limit int) ([]model.Post, error) {
	if len(authorIDs) == 0 {
		return []model.Post{}, nil
	}
	var posts []model.Post
	err := r.db.
		Table("posts").
		Select("posts.*, "+postAuthorSelect+", COUNT(comments.id) AS comment_count").
		Joins("LEFT JOIN users ON users.id = posts.author_id").
		Joins("LEFT JOIN comments ON comments.post_id = posts.id AND comments.status = ?", "normal").
		Where("posts.status = ? AND posts.author_id IN ?", "published", authorIDs).
		Group("posts.id").
		Order("posts.publish_at DESC").
		Limit(limit).
		Find(&posts).Error
	return posts, err
}
