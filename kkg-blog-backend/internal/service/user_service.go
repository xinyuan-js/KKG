package service

import (
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
