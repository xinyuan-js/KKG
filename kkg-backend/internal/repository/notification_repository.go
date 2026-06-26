package repository

import (
	"errors"
	"time"

	"kkg-backend/internal/model"

	"gorm.io/gorm"
)

type NotificationRepository struct {
	db *gorm.DB
}

const notificationActorSelect = "CASE WHEN users.status = -1 THEN '用户已删除或注销' ELSE users.username END AS actor_name, CASE WHEN users.status = -1 THEN '' ELSE users.avatar_url END AS actor_avatar_url"

func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Create(n *model.Notification) error {
	return r.db.Create(n).Error
}

func (r *NotificationRepository) ListByReceiver(receiverID uint64, limit int) ([]model.Notification, error) {
	var out []model.Notification
	err := r.db.
		Table("notifications").
		Select(`
notifications.*,
`+notificationActorSelect+`,
posts.title AS post_title,
comments.content AS comment_content
`).
		Joins("LEFT JOIN users ON users.id = notifications.actor_id").
		Joins("LEFT JOIN posts ON posts.id = notifications.post_id").
		Joins("LEFT JOIN comments ON comments.id = notifications.comment_id").
		Where("notifications.receiver_id = ?", receiverID).
		Order("notifications.is_read ASC").
		Order("notifications.created_at DESC").
		Limit(limit).
		Find(&out).Error
	return out, err
}

func (r *NotificationRepository) CountUnreadByReceiver(receiverID uint64) (int64, error) {
	var count int64
	err := r.db.Model(&model.Notification{}).
		Where("receiver_id = ? AND is_read = ?", receiverID, false).
		Count(&count).Error
	return count, err
}

func (r *NotificationRepository) GetByID(id uint64) (*model.Notification, error) {
	var n model.Notification
	err := r.db.First(&n, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *NotificationRepository) MarkRead(id uint64) error {
	now := time.Now()
	return r.db.Model(&model.Notification{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": &now,
		}).Error
}
