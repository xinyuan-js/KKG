package service

import (
	"crypto/md5"
	"encoding/hex"
	"time"

	"kkg-backend/internal/oj/common"
	"kkg-backend/internal/oj/model/entity"

	"gorm.io/gorm"
)

const salt = "yupi"

type UserService struct {
	db *gorm.DB
}

type sharedUser struct {
	ID           int64     `gorm:"column:id"`
	Username     string    `gorm:"column:username"`
	Email        string    `gorm:"column:email"`
	AvatarURL    string    `gorm:"column:avatar_url"`
	PasswordHash string    `gorm:"column:password_hash"`
	Role         string    `gorm:"column:role"`
	Status       int8      `gorm:"column:status"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (sharedUser) TableName() string { return "users" }

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

func (s *UserService) GetByID(userID int64) (*entity.User, error) {
	var u entity.User
	if err := s.db.Where("id = ? AND isDelete = 0", userID).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *UserService) IsAdmin(user *entity.User) bool {
	return user != nil && (user.UserRole == common.AdminRole || user.UserRole == common.SuperAdminRole)
}

func (s *UserService) IsSuperAdmin(user *entity.User) bool {
	return user != nil && user.UserRole == common.SuperAdminRole
}

func md5Text(text string) string {
	sum := md5.Sum([]byte(text))
	return hex.EncodeToString(sum[:])
}

func validUserRole(role string) bool {
	return role == common.DefaultRole || role == common.AdminRole || role == common.SuperAdminRole || role == common.BanRole
}
