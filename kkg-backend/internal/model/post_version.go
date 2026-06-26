package model

import "time"

type PostVersion struct {
	ID          uint64     `gorm:"primaryKey" json:"id"`
	PostID      uint64     `gorm:"not null;index" json:"post_id"`
	Version     int        `gorm:"not null;index" json:"version"`
	DraftNote   string     `gorm:"size:1024;not null;default:''" json:"draft_note"`
	Title       string     `gorm:"size:255;not null" json:"title"`
	Summary     string     `gorm:"size:512;not null;default:''" json:"summary"`
	Tags        StringList `gorm:"type:json;not null" json:"tags"`
	RawContent  string     `gorm:"type:longtext;not null" json:"raw_content"`
	HTMLContent string     `gorm:"type:longtext;not null" json:"html_content"`
	Status      string     `gorm:"size:32;not null" json:"status"`
	Visibility  string     `gorm:"size:16;not null" json:"visibility"`
	PublishAt   *time.Time `json:"publish_at"`
	OperatorID  uint64     `gorm:"not null;index" json:"operator_id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (PostVersion) TableName() string {
	return "post_versions"
}
