package model

import "time"

type Comment struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	PostID    uint64    `gorm:"not null;index" json:"post_id"`
	UserID    uint64    `gorm:"not null;index" json:"user_id"`
	ParentID  *uint64   `gorm:"index" json:"parent_id,omitempty"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	Status    string    `gorm:"size:16;not null;default:normal;index" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	AuthorName      string `gorm:"->;column:author_name" json:"author_name,omitempty"`
	AuthorAvatarURL string `gorm:"->;column:author_avatar_url" json:"author_avatar_url,omitempty"`
}

func (Comment) TableName() string {
	return "comments"
}
