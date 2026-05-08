package model

import (
	"time"

	"gorm.io/gorm"
)

type Post struct {
	ID               uint64         `gorm:"primaryKey" json:"id"`
	AuthorID         uint64         `gorm:"not null;index" json:"author_id"`
	Version          int            `gorm:"not null;default:0" json:"version"`
	PublishedVersion int            `gorm:"not null;default:0;index" json:"published_version"`
	Title            string         `gorm:"size:255;not null" json:"title"`
	Slug             string         `gorm:"size:255;uniqueIndex;not null" json:"slug"`
	Summary          string         `gorm:"size:512;not null;default:''" json:"summary"`
	Tags             StringList     `gorm:"type:json;not null" json:"tags"`
	RawContent       string         `gorm:"type:longtext;not null" json:"raw_content"`
	HTMLContent      string         `gorm:"type:longtext;not null" json:"html_content"`
	Status           string         `gorm:"size:32;not null;default:draft;index" json:"status"`
	Visibility       string         `gorm:"size:16;not null;default:public" json:"visibility"`
	PublishAt        *time.Time     `json:"publish_at"`
	AuthorName       string         `gorm:"->;column:author_name" json:"author_name,omitempty"`
	AuthorAvatarURL  string         `gorm:"->;column:author_avatar_url" json:"author_avatar_url,omitempty"`
	CommentCount     int64          `gorm:"->;column:comment_count" json:"comment_count,omitempty"`
	FeedScore        float64        `gorm:"-" json:"feed_score,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Post) TableName() string {
	return "posts"
}
