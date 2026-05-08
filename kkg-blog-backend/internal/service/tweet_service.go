package service

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"awesomeProject/internal/model"
	"awesomeProject/internal/repository"
	"awesomeProject/internal/search"

	"gorm.io/gorm"
)

const maxTweetLength = 280

type TweetService struct {
	tweets  *repository.TweetRepository
	es      *search.Client
	esIndex string
}

func NewTweetService(tweets *repository.TweetRepository, es *search.Client, esIndex string) *TweetService {
	return &TweetService{tweets: tweets, es: es, esIndex: esIndex}
}

func (s *TweetService) Create(authorID uint64, content string) (*model.Tweet, error) {
	content = strings.TrimSpace(content)
	if authorID == 0 {
		return nil, errors.New("invalid user context")
	}
	if content == "" {
		return nil, errors.New("content is required")
	}
	if len([]rune(content)) > maxTweetLength {
		return nil, errors.New("content too long, max 280 characters")
	}

	var tweet *model.Tweet
	err := s.tweets.DB().Transaction(func(tx *gorm.DB) error {
		txRepo := repository.NewTweetRepository(tx)
		t := &model.Tweet{AuthorID: authorID, Content: content}
		if err := txRepo.Create(t); err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := s.es.Index(ctx, s.esIndex, strconv.FormatUint(t.ID, 10), map[string]interface{}{
			"id":         t.ID,
			"author_id":  t.AuthorID,
			"content":    t.Content,
			"created_at": t.CreatedAt.Format(time.RFC3339),
		}); err != nil {
			return err
		}
		tweet = t
		return nil
	})
	if err != nil {
		return nil, err
	}
	return tweet, nil
}

type TweetSearchItem struct {
	ID        uint64  `json:"id"`
	AuthorID  uint64  `json:"author_id"`
	Content   string  `json:"content"`
	CreatedAt string  `json:"created_at"`
	Score     float64 `json:"score"`
}

func (s *TweetService) Search(query string, from int, size int) ([]TweetSearchItem, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("query is required")
	}
	if from < 0 {
		from = 0
	}
	if size <= 0 || size > 100 {
		size = 20
	}
	esQuery := map[string]interface{}{
		"from": from,
		"size": size,
		"sort": []map[string]interface{}{
			{"_score": map[string]string{"order": "desc"}},
			{"created_at": map[string]string{"order": "desc"}},
		},
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  query,
				"fields": []string{"content"},
			},
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	hits, err := s.es.Search(ctx, s.esIndex, esQuery)
	if err != nil {
		return nil, err
	}
	items := make([]TweetSearchItem, 0, len(hits))
	for _, hit := range hits {
		var src struct {
			ID        uint64 `json:"id"`
			AuthorID  uint64 `json:"author_id"`
			Content   string `json:"content"`
			CreatedAt string `json:"created_at"`
		}
		if err := json.Unmarshal(hit.Source, &src); err != nil {
			continue
		}
		items = append(items, TweetSearchItem{
			ID:        src.ID,
			AuthorID:  src.AuthorID,
			Content:   src.Content,
			CreatedAt: src.CreatedAt,
			Score:     hit.Score,
		})
	}
	return items, nil
}
