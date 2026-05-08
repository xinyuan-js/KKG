package model

import "time"

type Notification struct {
	ID              uint64     `gorm:"primaryKey" json:"id"`
	ReceiverID      uint64     `gorm:"not null;index" json:"receiver_id"`
	ActorID         uint64     `gorm:"not null;index" json:"actor_id"`
	PostID          uint64     `gorm:"not null;index" json:"post_id"`
	CommentID       uint64     `gorm:"not null;index" json:"comment_id"`
	ParentCommentID *uint64    `gorm:"index" json:"parent_comment_id,omitempty"`
	Type            string     `gorm:"size:32;not null;index" json:"type"`
	IsRead          bool       `gorm:"not null;default:false;index" json:"is_read"`
	ReadAt          *time.Time `json:"read_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`

	ActorName      string `gorm:"->;column:actor_name" json:"actor_name,omitempty"`
	ActorAvatarURL string `gorm:"->;column:actor_avatar_url" json:"actor_avatar_url,omitempty"`
	PostTitle      string `gorm:"->;column:post_title" json:"post_title,omitempty"`
	CommentContent string `gorm:"->;column:comment_content" json:"comment_content,omitempty"`
}

func (Notification) TableName() string {
	return "notifications"
}
