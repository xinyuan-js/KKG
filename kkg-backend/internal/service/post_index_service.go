package service

import (
	"kkg-backend/internal/model"
	"context"
	"strconv"
	"time"
)

func (s *PostService) indexPost(post *model.Post) {
	if s.es == nil || post == nil || post.Status != "published" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	doc := map[string]interface{}{
		"id":         post.ID,
		"author_id":  post.AuthorID,
		"title":      post.Title,
		"summary":    post.Summary,
		"tags":       post.Tags,
		"updated_at": post.UpdatedAt.Format(time.RFC3339),
	}
	if post.PublishAt != nil {
		doc["publish_at"] = post.PublishAt.Format(time.RFC3339)
	}
	_ = s.es.Index(ctx, s.esPostIndex, strconv.FormatUint(post.ID, 10), doc)
}

func (s *PostService) deletePostFromIndex(postID uint64) {
	if s.es == nil || postID == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = s.es.Delete(ctx, s.esPostIndex, strconv.FormatUint(postID, 10))
}
