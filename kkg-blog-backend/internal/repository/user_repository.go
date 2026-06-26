package repository

import (
	"errors"
	"strings"

	"awesomeProject/internal/model"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepository) GetByUsernameOrEmail(account string) (*model.User, error) {
	var user model.User
	err := r.db.Where("username = ? OR email = ?", account, account).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByID(id uint64) (*model.User, error) {
	var user model.User
	err := r.db.First(&user, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) ExistsByUsernameOrEmail(username string, email string) (bool, error) {
	var count int64
	err := r.db.Model(&model.User{}).
		Where("username = ? OR email = ?", username, email).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *UserRepository) ExistsByUsernameOrEmailExcludeID(username string, email string, excludeID uint64) (bool, error) {
	var count int64
	err := r.db.Model(&model.User{}).
		Where("(username = ? OR email = ?) AND id <> ?", username, email, excludeID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *UserRepository) Update(user *model.User) error {
	return r.db.Save(user).Error
}

func (r *UserRepository) DeleteByID(id uint64) error {
	return r.db.Delete(&model.User{}, id).Error
}

func (r *UserRepository) SearchByKeywordExcludeID(keyword string, excludeID uint64, limit int) ([]model.User, error) {
	var users []model.User
	kw := "%" + keyword + "%"
	q := r.db.Model(&model.User{}).Where("(username LIKE ? OR email LIKE ?) AND status <> -1", kw, kw)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	err := q.Order("id DESC").Limit(limit).Find(&users).Error
	return users, err
}

func (r *UserRepository) SuggestUsernamesExcludeID(keyword string, excludeID uint64, limit int) ([]string, error) {
	var usernames []string
	kw := "%" + keyword + "%"
	q := r.db.Model(&model.User{}).Where("username LIKE ? AND status <> -1", kw)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	err := q.Order("id DESC").Limit(limit).Pluck("username", &usernames).Error
	return usernames, err
}

func (r *UserRepository) ListForAdmin(keyword string, role string, statusFilter string, page int, pageSize int) ([]model.User, int64, error) {
	q := r.db.Model(&model.User{})
	kw := strings.TrimSpace(keyword)
	if kw != "" {
		like := "%" + kw + "%"
		q = q.Where("username LIKE ? OR email LIKE ?", like, like)
	}
	rv := strings.TrimSpace(role)
	if rv != "" {
		q = q.Where("role = ?", rv)
	}
	switch strings.TrimSpace(statusFilter) {
	case "active":
		q = q.Where("status = 1")
	case "disabled":
		q = q.Where("status = 0")
	case "hidden", "deleted":
		q = q.Where("status = -1")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []model.User
	err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&users).Error
	return users, total, err
}
