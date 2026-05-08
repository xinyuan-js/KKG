package model

import "time"

type Tweet struct {
	ID        uint64    `json:"id" gorm:"primaryKey;autoIncrement"`
	AuthorID  uint64    `json:"author_id" gorm:"index;not null"`
	Content   string    `json:"content" gorm:"type:text;not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
