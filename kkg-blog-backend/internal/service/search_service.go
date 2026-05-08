package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"awesomeProject/internal/search"
)

type SearchService struct {
	es        *search.Client
	postIndex string
	userIndex string
}

func NewSearchService(es *search.Client, postIndex string, userIndex string) *SearchService {
	return &SearchService{es: es, postIndex: postIndex, userIndex: userIndex}
}

type PostSearchItem struct {
	ID       uint64  `json:"id"`
	AuthorID uint64  `json:"author_id"`
	Title    string  `json:"title"`
	Summary  string  `json:"summary"`
	Status   string  `json:"status"`
	Score    float64 `json:"score"`
}

func (s *SearchService) SearchPosts(keyword string, limit int) ([]PostSearchItem, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, errors.New("query is required")
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	query := map[string]interface{}{
		"size": limit,
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  keyword,
				"fields": []string{"title^2", "summary", "tags^1.5"},
			},
		},
		"sort": []map[string]interface{}{
			{"_score": map[string]string{"order": "desc"}},
			{"publish_at": map[string]string{"order": "desc"}},
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	hits, err := s.es.Search(ctx, s.postIndex, query)
	if err != nil {
		return nil, err
	}
	out := make([]PostSearchItem, 0, len(hits))
	for _, hit := range hits {
		var src struct {
			ID       uint64 `json:"id"`
			AuthorID uint64 `json:"author_id"`
			Title    string `json:"title"`
			Summary  string `json:"summary"`
		}
		if err := json.Unmarshal(hit.Source, &src); err != nil {
			continue
		}
		out = append(out, PostSearchItem{
			ID:       src.ID,
			AuthorID: src.AuthorID,
			Title:    src.Title,
			Summary:  src.Summary,
			Status:   "published",
			Score:    hit.Score,
		})
	}
	return out, nil
}

type UserSearchItem struct {
	ID        uint64  `json:"id"`
	Username  string  `json:"username"`
	Email     string  `json:"email"`
	AvatarURL string  `json:"avatar_url"`
	Score     float64 `json:"score"`
}

func (s *SearchService) SearchUsers(keyword string, excludeID uint64, limit int) ([]UserSearchItem, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, errors.New("query is required")
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	boolQuery := map[string]interface{}{
		"must": []map[string]interface{}{
			{
				"multi_match": map[string]interface{}{
					"query":  keyword,
					"fields": []string{"username^3"},
				},
			},
		},
	}
	if excludeID > 0 {
		boolQuery["must_not"] = []map[string]interface{}{
			{"term": map[string]interface{}{"id": excludeID}},
		}
	}
	query := map[string]interface{}{
		"size":  limit,
		"query": map[string]interface{}{"bool": boolQuery},
		"sort": []map[string]interface{}{
			{"_score": map[string]string{"order": "desc"}},
			{"id": map[string]string{"order": "desc"}},
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	hits, err := s.es.Search(ctx, s.userIndex, query)
	if err != nil {
		return nil, err
	}
	out := make([]UserSearchItem, 0, len(hits))
	for _, hit := range hits {
		var src UserSearchItem
		if err := json.Unmarshal(hit.Source, &src); err != nil {
			continue
		}
		src.Score = hit.Score
		out = append(out, src)
	}
	return out, nil
}

func (s *SearchService) SuggestPosts(keyword string, limit int) ([]string, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return []string{}, nil
	}
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	query := map[string]interface{}{
		"size": limit,
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":    keyword,
				"type":     "bool_prefix",
				"fields":   []string{"title", "title._2gram", "title._3gram", "tags"},
				"operator": "or",
			},
		},
		"_source": []string{"title"},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	hits, err := s.es.Search(ctx, s.postIndex, query)
	if err != nil {
		return nil, err
	}
	items := make([]string, 0, len(hits))
	seen := make(map[string]struct{}, len(hits))
	for _, hit := range hits {
		var src struct {
			Title string `json:"title"`
		}
		if err := json.Unmarshal(hit.Source, &src); err != nil {
			continue
		}
		t := strings.TrimSpace(src.Title)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		items = append(items, t)
	}
	return items, nil
}

func (s *SearchService) SuggestUsers(keyword string, excludeID uint64, limit int) ([]string, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return []string{}, nil
	}
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	boolQuery := map[string]interface{}{
		"must": []map[string]interface{}{
			{
				"multi_match": map[string]interface{}{
					"query":    keyword,
					"type":     "bool_prefix",
					"fields":   []string{"username", "username._2gram", "username._3gram"},
					"operator": "or",
				},
			},
		},
	}
	if excludeID > 0 {
		boolQuery["must_not"] = []map[string]interface{}{
			{"term": map[string]interface{}{"id": excludeID}},
		}
	}
	query := map[string]interface{}{
		"size":    limit,
		"query":   map[string]interface{}{"bool": boolQuery},
		"_source": []string{"username"},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	hits, err := s.es.Search(ctx, s.userIndex, query)
	if err != nil {
		return nil, err
	}
	items := make([]string, 0, len(hits))
	seen := make(map[string]struct{}, len(hits))
	for _, hit := range hits {
		var src struct {
			Username string `json:"username"`
		}
		if err := json.Unmarshal(hit.Source, &src); err != nil {
			continue
		}
		u := strings.TrimSpace(src.Username)
		if u == "" {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		items = append(items, u)
	}
	return items, nil
}
