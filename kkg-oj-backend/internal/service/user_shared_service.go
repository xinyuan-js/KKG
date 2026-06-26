package service

import (
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"strings"
	"yuoj-go-backend/internal/common"
	"yuoj-go-backend/internal/model/entity"
)

func (s *UserService) EnsureFromSharedUserID(sharedID int64) (*entity.User, error) {
	if sharedID <= 0 {
		return nil, errors.New("invalid shared user id")
	}
	var shared sharedUser
	if err := s.db.Where("id = ? AND status = 1", sharedID).First(&shared).Error; err != nil {
		return nil, err
	}
	var u entity.User
	err := s.db.Where("userAccount = ? AND isDelete = 0", shared.Username).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		u = entity.User{
			UserAccount:  shared.Username,
			UserPassword: md5Text(salt + fmt.Sprintf("sso-%d", sharedID)),
			UserName:     shared.Username,
			UserAvatar:   shared.AvatarURL,
			UserRole:     shared.Role,
		}
		if createErr := s.db.Create(&u).Error; createErr != nil {
			return nil, createErr
		}
		return &u, nil
	}
	if err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	if shared.Username != "" && u.UserName != shared.Username {
		u.UserName = shared.Username
		updates["userName"] = shared.Username
	}
	if u.UserAvatar != shared.AvatarURL {
		u.UserAvatar = shared.AvatarURL
		updates["userAvatar"] = shared.AvatarURL
	}
	if shared.Role != "" && u.UserRole != shared.Role {
		u.UserRole = shared.Role
		updates["userRole"] = shared.Role
	}
	if len(updates) > 0 {
		_ = s.db.Model(&entity.User{}).Where("id = ?", u.ID).Updates(updates).Error
	}
	return &u, nil
}

func (s *UserService) SharedUserIDByAccount(account string) (int64, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return 0, nil
	}
	var shared sharedUser
	if err := s.db.Select("id").Where("username = ? AND status <> -1", account).First(&shared).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return shared.ID, nil
}

func (s *UserService) ensureSharedUser(account, password string, u *entity.User) error {
	if u == nil || strings.TrimSpace(account) == "" || strings.TrimSpace(password) == "" {
		return nil
	}
	var shared sharedUser
	err := s.db.Where("username = ? OR email = ?", account, account).First(&shared).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	email := account
	if !strings.Contains(email, "@") {
		email = account + "@kkgoj.local"
	}
	hash, hashErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if hashErr != nil {
		return hashErr
	}
	nu := &sharedUser{
		Username:     account,
		Email:        email,
		AvatarURL:    u.UserAvatar,
		PasswordHash: string(hash),
		Role:         u.UserRole,
		Status:       1,
	}
	return s.db.Create(nu).Error
}

func (s *UserService) loginFromSharedUser(account, password string) (*entity.User, error) {
	var shared sharedUser
	if err := s.db.Where("(username = ? OR email = ?) AND status = 1", account, account).First(&shared).Error; err != nil {
		return nil, err
	}
	if strings.EqualFold(shared.Role, common.BanRole) {
		return nil, common.NewBizError(common.ForbiddenError, "该用户已被封，禁止登录")
	}
	if bcrypt.CompareHashAndPassword([]byte(shared.PasswordHash), []byte(password)) != nil {
		return nil, errors.New("invalid credentials")
	}
	var u entity.User
	err := s.db.Where("userAccount = ? AND isDelete = 0", shared.Username).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		u = entity.User{
			UserAccount:  shared.Username,
			UserPassword: md5Text(salt + password),
			UserName:     shared.Username,
			UserAvatar:   shared.AvatarURL,
			UserRole:     shared.Role,
		}
		if createErr := s.db.Create(&u).Error; createErr != nil {
			return nil, createErr
		}
		return &u, nil
	}
	if err != nil {
		return nil, err
	}
	if shared.Role != "" && u.UserRole != shared.Role {
		u.UserRole = shared.Role
		_ = s.db.Model(&entity.User{}).Where("id = ?", u.ID).Update("userRole", shared.Role).Error
	}
	return &u, nil
}
