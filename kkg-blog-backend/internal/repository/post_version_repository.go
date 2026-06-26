package repository

import (
	"awesomeProject/internal/model"
	"errors"
	"gorm.io/gorm"
)

func (r *PostRepository) CreateVersion(version *model.PostVersion) error {
	return r.db.Create(version).Error
}

func (r *PostRepository) ListVersions(postID uint64, limit int) ([]model.PostVersion, error) {
	var versions []model.PostVersion
	err := r.db.
		Where("post_id = ?", postID).
		Order("CASE WHEN status = 'published' THEN 0 ELSE 1 END ASC").
		Order("updated_at DESC").
		Order("version DESC").
		Limit(limit).
		Find(&versions).Error
	return versions, err
}

func (r *PostRepository) GetVersion(postID uint64, version int) (*model.PostVersion, error) {
	var data model.PostVersion
	err := r.db.Where("post_id = ? AND version = ?", postID, version).First(&data).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *PostRepository) DeleteVersion(postID uint64, version int) error {
	return r.db.Where("post_id = ? AND version = ?", postID, version).Delete(&model.PostVersion{}).Error
}

func (r *PostRepository) CountVersions(postID uint64) (int64, error) {
	var count int64
	err := r.db.Model(&model.PostVersion{}).Where("post_id = ?", postID).Count(&count).Error
	return count, err
}

func (r *PostRepository) GetLatestVersion(postID uint64) (*model.PostVersion, error) {
	var data model.PostVersion
	err := r.db.Where("post_id = ?", postID).Order("version DESC").First(&data).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *PostRepository) SetAllDraftStatus(postID uint64, status string, publishAt interface{}) error {
	return r.db.Model(&model.PostVersion{}).Where("post_id = ?", postID).Updates(map[string]interface{}{
		"status":     status,
		"publish_at": publishAt,
	}).Error
}

func (r *PostRepository) UpdateDraftByVersion(postID uint64, version int, fields map[string]interface{}) error {
	return r.db.Model(&model.PostVersion{}).Where("post_id = ? AND version = ?", postID, version).Updates(fields).Error
}

func (r *PostRepository) UpdateAllVersionMeta(postID uint64, title string, summary string, tags model.StringList, operatorID uint64) error {
	return r.db.Model(&model.PostVersion{}).
		Where("post_id = ?", postID).
		Updates(map[string]interface{}{
			"title":       title,
			"summary":     summary,
			"tags":        tags,
			"operator_id": operatorID,
		}).Error
}

func (r *PostRepository) DeleteVersionsByPostID(postID uint64) error {
	return r.db.Where("post_id = ?", postID).Delete(&model.PostVersion{}).Error
}

func (r *PostRepository) DeletePost(post *model.Post) error {
	return r.db.Delete(post).Error
}
