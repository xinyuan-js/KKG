package service

import (
	"kkg-backend/internal/oj/common"
	"kkg-backend/internal/oj/model/entity"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"strings"
)

func (s *UserService) EnsureFromSharedUserID(sharedID int64) (*entity.User, error) {
	if sharedID <= 0 {
		return nil, errors.New("invalid shared user id")
	}
	var shared sharedUser
	if err := s.db.Where("id = ? AND status = 1", sharedID).First(&shared).Error; err != nil {
		return nil, err
	}
	if mapped, err := s.findLegacyOJUserBySharedID(sharedID); err != nil {
		return nil, err
	} else if mapped != nil {
		s.syncOJUserFromShared(mapped, &shared)
		return mapped, nil
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
	s.syncOJUserFromShared(&u, &shared)
	return &u, nil
}

func (s *UserService) findLegacyOJUserBySharedID(sharedID int64) (*entity.User, error) {
	var m struct {
		LegacyOJUserID int64 `gorm:"column:legacy_oj_user_id"`
	}
	err := s.db.Table("user_identity_map").
		Select("legacy_oj_user_id").
		Where("auth_user_id = ? AND source = ? AND legacy_oj_user_id IS NOT NULL", sharedID, "oj").
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var u entity.User
	err = s.db.Where("id = ? AND isDelete = 0", m.LegacyOJUserID).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *UserService) syncOJUserFromShared(u *entity.User, shared *sharedUser) {
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

func (s *UserService) EnsureSharedUserForOJUser(u *entity.User, passwordSeed string) (int64, error) {
	if u == nil || strings.TrimSpace(u.UserAccount) == "" {
		return 0, errors.New("invalid user")
	}
	account := strings.TrimSpace(u.UserAccount)
	var shared sharedUser
	err := s.db.Where("username = ? OR email = ?", account, account).First(&shared).Error
	if err == nil {
		updates := map[string]interface{}{}
		if shared.Status != 1 {
			updates["status"] = int8(1)
		}
		if strings.TrimSpace(u.UserRole) != "" && shared.Role != u.UserRole {
			updates["role"] = u.UserRole
		}
		if u.UserAvatar != "" && shared.AvatarURL != u.UserAvatar {
			updates["avatar_url"] = u.UserAvatar
		}
		if len(updates) > 0 {
			_ = s.db.Model(&sharedUser{}).Where("id = ?", shared.ID).Updates(updates).Error
		}
		return shared.ID, s.ensureIdentityMap(shared.ID, u.ID, account)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}

	email := account
	if !strings.Contains(email, "@") {
		email = account + "@kkgoj.local"
	}
	seed := strings.TrimSpace(passwordSeed)
	if seed == "" {
		seed = fmt.Sprintf("oj-user-%d", u.ID)
	}
	hash, hashErr := bcrypt.GenerateFromPassword([]byte(seed), bcrypt.DefaultCost)
	if hashErr != nil {
		return 0, hashErr
	}
	shared = sharedUser{
		Username:     account,
		Email:        email,
		AvatarURL:    u.UserAvatar,
		PasswordHash: string(hash),
		Role:         u.UserRole,
		Status:       1,
	}
	if strings.TrimSpace(shared.Role) == "" {
		shared.Role = common.DefaultRole
	}
	if err := s.db.Create(&shared).Error; err != nil {
		return 0, err
	}
	return shared.ID, s.ensureIdentityMap(shared.ID, u.ID, account)
}

func (s *UserService) ensureIdentityMap(sharedID int64, legacyOJUserID int64, account string) error {
	if sharedID <= 0 || legacyOJUserID <= 0 {
		return nil
	}
	return s.db.Exec(`
		INSERT INTO user_identity_map (auth_user_id, legacy_oj_user_id, legacy_oj_account, source)
		VALUES (?, ?, ?, 'oj')
		ON DUPLICATE KEY UPDATE
			auth_user_id = VALUES(auth_user_id),
			legacy_oj_account = VALUES(legacy_oj_account),
			updated_at = CURRENT_TIMESTAMP
	`, sharedID, legacyOJUserID, strings.TrimSpace(account)).Error
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
