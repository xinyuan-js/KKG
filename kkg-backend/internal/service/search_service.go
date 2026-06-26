package service

import (
	"kkg-backend/internal/search"
)

type SearchService struct {
	es        *search.Client
	postIndex string
	userIndex string
}

func NewSearchService(es *search.Client, postIndex string, userIndex string) *SearchService {
	return &SearchService{es: es, postIndex: postIndex, userIndex: userIndex}
}
