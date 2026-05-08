package service

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"yuoj-go-backend/internal/common"
	"yuoj-go-backend/internal/model/entity"

	"golang.org/x/crypto/bcrypt"
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

func (s *UserService) Register(account, password, checkPassword string) (int64, error) {
	if account == "" || password == "" || checkPassword == "" {
		return 0, common.NewBizError(common.ParamsError, "参数为空")
	}
	if len(account) < 4 || len(password) < 8 || len(checkPassword) < 8 {
		return 0, common.NewBizError(common.ParamsError, "参数错误")
	}
	if password != checkPassword {
		return 0, common.NewBizError(common.ParamsError, "两次输入的密码不一致")
	}
	var cnt int64
	if err := s.db.Model(&entity.User{}).Where("userAccount = ? AND isDelete = 0", account).Count(&cnt).Error; err != nil {
		return 0, err
	}
	if cnt > 0 {
		return 0, common.NewBizError(common.ParamsError, "账号重复")
	}
	u := &entity.User{
		UserAccount:  account,
		UserPassword: md5Text(salt + password),
		UserName:     account,
		UserRole:     common.DefaultRole,
	}
	if err := s.db.Create(u).Error; err != nil {
		return 0, err
	}
	_ = s.ensureSharedUser(account, password, u)
	return u.ID, nil
}

func (s *UserService) Login(account, password string) (*entity.User, error) {
	if len(account) < 4 || len(password) < 8 {
		return nil, common.NewBizError(common.ParamsError, "账号或密码错误")
	}
	var u entity.User
	err := s.db.Where("userAccount = ? AND userPassword = ? AND isDelete = 0", account, md5Text(salt+password)).First(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			shared, sharedErr := s.loginFromSharedUser(account, password)
			if sharedErr != nil {
				return nil, common.NewBizError(common.ParamsError, "用户不存在或密码错误")
			}
			return shared, nil
		}
		return nil, err
	}
	if strings.EqualFold(u.UserRole, common.BanRole) {
		return nil, common.NewBizError(common.ForbiddenError, "该用户已被封，禁止登录")
	}
	_ = s.ensureSharedUser(account, password, &u)
	return &u, nil
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
	// 同步主账号侧角色，确保超级管理员/管理员权限实时生效。
	if shared.Role != "" && u.UserRole != shared.Role {
		u.UserRole = shared.Role
		_ = s.db.Model(&entity.User{}).Where("id = ?", u.ID).Update("userRole", shared.Role).Error
	}
	return &u, nil
}

func (s *UserService) List(page, size int64) ([]entity.User, int64, error) {
	var total int64
	if err := s.db.Model(&entity.User{}).Where("isDelete = 0").Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []entity.User
	err := s.db.Where("isDelete = 0").Offset(int((page - 1) * size)).Limit(int(size)).Find(&list).Error
	return list, total, err
}

func (s *UserService) CreateByAdmin(u *entity.User) error {
	if u.UserRole == "" {
		u.UserRole = common.DefaultRole
	}
	if strings.TrimSpace(u.UserPassword) != "" {
		u.UserPassword = md5Text(salt + u.UserPassword)
	}
	return s.db.Create(u).Error
}

func (s *UserService) UpdateByID(u *entity.User) error {
	return s.db.Model(&entity.User{}).Where("id = ? AND isDelete = 0", u.ID).Updates(u).Error
}

func (s *UserService) SoftDelete(id int64) error {
	return s.db.Model(&entity.User{}).Where("id = ? AND isDelete = 0", id).Update("isDelete", 1).Error
}

func (s *UserService) UpdateByAdmin(actor *entity.User, patch *entity.User) error {
	if actor == nil || patch == nil || patch.ID <= 0 {
		return common.NewBizError(common.ParamsError, "请求参数错误")
	}
	if !s.IsAdmin(actor) {
		return common.NewBizError(common.NoAuthError, "无权限")
	}
	if patch.ID == actor.ID {
		return common.NewBizError(common.NoAuthError, "不能通过管理接口修改自己")
	}
	target, err := s.GetByID(patch.ID)
	if err != nil {
		return err
	}
	if target == nil {
		return common.NewBizError(common.NotFoundError, "用户不存在")
	}
	if target.UserRole == common.SuperAdminRole && !s.IsSuperAdmin(actor) {
		return common.NewBizError(common.NoAuthError, "无权限")
	}
	nextRole := strings.TrimSpace(patch.UserRole)
	if nextRole != "" {
		if !s.IsSuperAdmin(actor) {
			return common.NewBizError(common.NoAuthError, "仅超级管理员可修改用户权限")
		}
		if nextRole != common.DefaultRole && nextRole != common.AdminRole && nextRole != common.SuperAdminRole && nextRole != common.BanRole {
			return common.NewBizError(common.ParamsError, "请求参数错误")
		}
	}
	return s.UpdateByID(patch)
}

func (s *UserService) SoftDeleteBySuperAdmin(actor *entity.User, targetID int64) error {
	if actor == nil || targetID <= 0 {
		return common.NewBizError(common.ParamsError, "请求参数错误")
	}
	if !s.IsSuperAdmin(actor) {
		return common.NewBizError(common.NoAuthError, "仅超级管理员可操作")
	}
	if actor.ID == targetID {
		return common.NewBizError(common.NoAuthError, "不能删除自己")
	}
	target, err := s.GetByID(targetID)
	if err != nil {
		return err
	}
	if target == nil {
		return common.NewBizError(common.NotFoundError, "用户不存在")
	}
	if target.UserRole == common.SuperAdminRole {
		return common.NewBizError(common.NoAuthError, "不能删除超级管理员")
	}
	return s.SoftDelete(targetID)
}

func md5Text(text string) string {
	sum := md5.Sum([]byte(text))
	return hex.EncodeToString(sum[:])
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
