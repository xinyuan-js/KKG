package db

import (
	"yuoj-go-backend/internal/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func NewMySQL(cfg *config.Config) (*gorm.DB, error) {
	return gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{})
}
