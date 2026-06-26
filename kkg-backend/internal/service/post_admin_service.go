package service

import (
	"kkg-backend/internal/model"
)

func (s *PostService) ListMine(authorID uint64, limit int) ([]model.Post, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.posts.ListByAuthor(authorID, limit)
}

func (s *PostService) ListForAdmin(keyword string, status string, page int, pageSize int) ([]model.Post, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	return s.posts.ListForAdmin(keyword, status, page, pageSize)
}
