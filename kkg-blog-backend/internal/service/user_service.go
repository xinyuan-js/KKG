package service

import (
	"errors"
	"strings"

	"awesomeProject/internal/model"
	"awesomeProject/internal/repository"
)

type PublicUserPage struct {
	ID             uint64       `json:"id"`
	Username       string       `json:"username"`
	AvatarURL      string       `json:"avatar_url"`
	CreatedAt      string       `json:"created_at"`
	PublishedCount int64        `json:"published_count"`
	Posts          []model.Post `json:"posts"`
}

type UserService struct {
	users *repository.UserRepository
	posts *repository.PostRepository
}

func NewUserService(users *repository.UserRepository, posts *repository.PostRepository) *UserService {
	return &UserService{users: users, posts: posts}
}

func (s *UserService) GetPublicPage(userID uint64, limit int) (*PublicUserPage, error) {
	if userID == 0 {
		return nil, errors.New("invalid user id")
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	user, err := s.users.GetByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}
	if user.Status == -1 {
		return nil, errors.New("user not found")
	}

	count, err := s.posts.CountPublishedByAuthor(userID)
	if err != nil {
		return nil, err
	}

	posts, err := s.posts.ListPublishedByAuthor(userID, limit)
	if err != nil {
		return nil, err
	}

	return &PublicUserPage{
		ID:             user.ID,
		Username:       user.Username,
		AvatarURL:      user.AvatarURL,
		CreatedAt:      user.CreatedAt.Format("2006-01-02 15:04:05"),
		PublishedCount: count,
		Posts:          posts,
	}, nil
}

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

func (s *UserService) SearchPublicUsers(keyword string, excludeID uint64, limit int) ([]model.User, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	return s.users.SearchByKeywordExcludeID(keyword, excludeID, limit)
}

func (s *UserService) SuggestPublicUsernames(keyword string, excludeID uint64, limit int) ([]string, error) {
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	return s.users.SuggestUsernamesExcludeID(keyword, excludeID, limit)
}
