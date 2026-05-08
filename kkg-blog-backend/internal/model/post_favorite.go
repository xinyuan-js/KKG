package model

import "time"

type PostFavorite struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	PostID    uint64    `gorm:"not null;uniqueIndex:idx_post_fav_post_user,priority:1;index" json:"post_id"`
	UserID    uint64    `gorm:"not null;uniqueIndex:idx_post_fav_post_user,priority:2;index" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (PostFavorite) TableName() string {
	return "post_favorites"
}
