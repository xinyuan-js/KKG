package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type PostSearchItem struct {
	ID       uint64   `json:"id"`
	AuthorID uint64   `json:"author_id"`
	Title    string   `json:"title"`
	Summary  string   `json:"summary"`
	Status   string   `json:"status"`
	Tags     []string `json:"tags,omitempty"`
	Score    float64  `json:"score"`
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
			ID       uint64   `json:"id"`
			AuthorID uint64   `json:"author_id"`
			Title    string   `json:"title"`
			Summary  string   `json:"summary"`
			Tags     []string `json:"tags"`
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
			Tags:     src.Tags,
			Score:    hit.Score,
		})
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
