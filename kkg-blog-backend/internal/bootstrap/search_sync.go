package bootstrap

import (
	"awesomeProject/internal/config"
	"awesomeProject/internal/model"
	"awesomeProject/internal/search"
	"context"
	"errors"
	"gorm.io/gorm"
	"strconv"
	"strings"
	"time"
)

func syncSearchData(db *gorm.DB, es *search.Client, cfg *config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := ensureSearchIndices(ctx, es, cfg); err != nil {
		return err
	}

	var posts []model.Post
	if err := db.Where("status = ?", "published").Find(&posts).Error; err != nil {
		return err
	}
	for _, p := range posts {
		doc := map[string]interface{}{
			"id":         p.ID,
			"author_id":  p.AuthorID,
			"title":      p.Title,
			"summary":    p.Summary,
			"tags":       p.Tags,
			"updated_at": p.UpdatedAt.Format(time.RFC3339),
		}
		if p.PublishAt != nil {
			doc["publish_at"] = p.PublishAt.Format(time.RFC3339)
		}
		if err := es.Index(ctx, cfg.ESPostIndex, strconv.FormatUint(p.ID, 10), doc); err != nil {
			return err
		}
	}
	var users []model.User
	if err := db.Find(&users).Error; err != nil {
		return err
	}
	for _, u := range users {
		doc := map[string]interface{}{
			"id":         u.ID,
			"username":   u.Username,
			"email":      u.Email,
			"avatar_url": u.AvatarURL,
			"updated_at": u.UpdatedAt.Format(time.RFC3339),
		}
		if err := es.Index(ctx, cfg.ESUserIndex, strconv.FormatUint(u.ID, 10), doc); err != nil {
			return err
		}
	}
	return nil
}

func ensureSearchIndices(ctx context.Context, es *search.Client, cfg *config.Config) error {
	postIndex := strings.TrimSpace(cfg.ESPostIndex)
	userIndex := strings.TrimSpace(cfg.ESUserIndex)
	if postIndex == "" || userIndex == "" {
		return errors.New("elasticsearch post/user index is required")
	}
	if err := es.CreateIndex(ctx, postIndex, map[string]interface{}{
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"id":         map[string]interface{}{"type": "long"},
				"author_id":  map[string]interface{}{"type": "long"},
				"title":      map[string]interface{}{"type": "search_as_you_type", "analyzer": "ik_max_word", "search_analyzer": "ik_smart"},
				"summary":    map[string]interface{}{"type": "text", "analyzer": "ik_max_word", "search_analyzer": "ik_smart"},
				"tags":       map[string]interface{}{"type": "text", "analyzer": "ik_max_word", "search_analyzer": "ik_smart", "fields": map[string]interface{}{"keyword": map[string]interface{}{"type": "keyword", "ignore_above": 256}}},
				"publish_at": map[string]interface{}{"type": "date"},
				"updated_at": map[string]interface{}{"type": "date"},
			},
		},
	}); err != nil {
		return err
	}
	return es.CreateIndex(ctx, userIndex, map[string]interface{}{
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"id":         map[string]interface{}{"type": "long"},
				"username":   map[string]interface{}{"type": "search_as_you_type", "analyzer": "ik_max_word", "search_analyzer": "ik_smart"},
				"email":      map[string]interface{}{"type": "keyword"},
				"avatar_url": map[string]interface{}{"type": "keyword", "ignore_above": 512},
				"updated_at": map[string]interface{}{"type": "date"},
			},
		},
	})
}
