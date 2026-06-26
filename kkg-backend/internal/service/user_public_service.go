package service

import (
	"errors"

	"kkg-backend/internal/model"
)

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
