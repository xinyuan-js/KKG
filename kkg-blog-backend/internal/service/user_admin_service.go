package service

import (
	"awesomeProject/internal/model"
	"errors"
	"strings"
)

func (s *UserService) ListForAdmin(keyword string, role string, statusFilter string, page int, pageSize int) ([]model.User, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	return s.users.ListForAdmin(keyword, role, statusFilter, page, pageSize)
}

func (s *UserService) UpdateRoleStatusByAdmin(actorRole string, actorID uint64, targetID uint64, role string, status int8) error {
	if targetID == 0 || actorID == 0 {
		return errors.New("invalid user id")
	}
	if actorID == targetID {
		return errors.New("cannot update self")
	}
	role = strings.TrimSpace(role)
	if role != "user" && role != "admin" && role != "super_admin" {
		return errors.New("invalid role")
	}
	if status != -1 && status != 0 && status != 1 {
		return errors.New("invalid status")
	}
	target, err := s.users.GetByID(targetID)
	if err != nil {
		return err
	}
	if target == nil {
		return errors.New("user not found")
	}
	if target.Role == "super_admin" && actorRole != "super_admin" {
		return errors.New("forbidden")
	}
	if role == "super_admin" && actorRole != "super_admin" {
		return errors.New("forbidden")
	}
	if role != target.Role && actorRole != "super_admin" {
		return errors.New("forbidden")
	}
	if (status == -1 || target.Status == -1) && actorRole != "super_admin" {
		return errors.New("forbidden")
	}
	target.Role = role
	target.Status = status
	return s.users.Update(target)
}

func (s *UserService) DeleteByAdmin(actorRole string, actorID uint64, targetID uint64) error {
	if actorRole != "super_admin" {
		return errors.New("forbidden")
	}
	if targetID == 0 || actorID == 0 {
		return errors.New("invalid user id")
	}
	if actorID == targetID {
		return errors.New("cannot delete self")
	}
	target, err := s.users.GetByID(targetID)
	if err != nil {
		return err
	}
	if target == nil {
		return errors.New("user not found")
	}
	if target.Role == "super_admin" {
		return errors.New("forbidden")
	}
	target.Status = -1
	return s.users.Update(target)
}
