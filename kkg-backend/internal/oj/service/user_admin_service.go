package service

import (
	"strings"
	"kkg-backend/internal/oj/common"
	"kkg-backend/internal/oj/model/entity"
)

func (s *UserService) List(page, size int64) ([]entity.User, int64, error) {
	var total int64
	if err := s.db.Model(&entity.User{}).Where("isDelete = 0").Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []entity.User
	err := s.db.Where("isDelete = 0").Offset(int((page - 1) * size)).Limit(int(size)).Find(&list).Error
	return list, total, err
}

func (s *UserService) CreateByAdmin(actor *entity.User, u *entity.User) error {
	if actor == nil || u == nil {
		return common.NewBizError(common.ParamsError, "请求参数错误")
	}
	if !s.IsAdmin(actor) {
		return common.NewBizError(common.NoAuthError, "无权限")
	}
	u.UserRole = strings.TrimSpace(u.UserRole)
	if u.UserRole == "" {
		u.UserRole = common.DefaultRole
	}
	if !validUserRole(u.UserRole) {
		return common.NewBizError(common.ParamsError, "请求参数错误")
	}
	if u.UserRole != common.DefaultRole && !s.IsSuperAdmin(actor) {
		return common.NewBizError(common.NoAuthError, "仅超级管理员可创建管理员账号")
	}
	if strings.TrimSpace(u.UserPassword) != "" {
		u.UserPassword = md5Text(salt + u.UserPassword)
	}
	return s.db.Create(u).Error
}

func (s *UserService) UpdateByID(u *entity.User) error {
	return s.db.Model(&entity.User{}).Where("id = ? AND isDelete = 0", u.ID).Updates(u).Error
}

func (s *UserService) UpdateProfile(userID int64, patch *entity.User) error {
	if userID <= 0 || patch == nil {
		return common.NewBizError(common.ParamsError, "请求参数错误")
	}
	updates := map[string]interface{}{}
	if strings.TrimSpace(patch.UserName) != "" {
		updates["userName"] = strings.TrimSpace(patch.UserName)
	}
	if patch.UserAvatar != "" {
		updates["userAvatar"] = patch.UserAvatar
	}
	if patch.UserProfile != "" {
		updates["userProfile"] = patch.UserProfile
	}
	if len(updates) == 0 {
		return nil
	}
	return s.db.Model(&entity.User{}).Where("id = ? AND isDelete = 0", userID).Updates(updates).Error
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
		if !validUserRole(nextRole) {
			return common.NewBizError(common.ParamsError, "请求参数错误")
		}
	}
	updates := map[string]interface{}{}
	if strings.TrimSpace(patch.UserName) != "" {
		updates["userName"] = strings.TrimSpace(patch.UserName)
	}
	if patch.UserAvatar != "" {
		updates["userAvatar"] = patch.UserAvatar
	}
	if patch.UserProfile != "" {
		updates["userProfile"] = patch.UserProfile
	}
	if nextRole != "" {
		updates["userRole"] = nextRole
	}
	if len(updates) == 0 {
		return nil
	}
	return s.db.Model(&entity.User{}).Where("id = ? AND isDelete = 0", patch.ID).Updates(updates).Error
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
