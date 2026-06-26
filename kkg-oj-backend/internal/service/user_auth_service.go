package service

import (
	"errors"
	"gorm.io/gorm"
	"strings"
	"yuoj-go-backend/internal/common"
	"yuoj-go-backend/internal/model/entity"
)

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
