package service

import (
	"awesomeProject/internal/model"
	"awesomeProject/internal/repository"
	"awesomeProject/internal/search"
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"strings"
	"time"
)

type PostService struct {
	posts       *repository.PostRepository
	redis       *redis.Client
	redCtx      context.Context
	es          *search.Client
	esPostIndex string
}

func NewPostService(posts *repository.PostRepository, redisClient *redis.Client, es *search.Client, esPostIndex string) *PostService {
	return &PostService{
		posts:       posts,
		redis:       redisClient,
		redCtx:      context.Background(),
		es:          es,
		esPostIndex: esPostIndex,
	}
}

func (s *PostService) makeUniqueSlug(baseSlug string, authorID uint64) (string, error) {
	base := fmt.Sprintf("%s-u%d", baseSlug, authorID)
	candidate := base
	for i := 0; i < 1000; i++ {
		exists, err := s.posts.ExistsSlug(candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i+2)
	}
	return fmt.Sprintf("%s-%d", base, time.Now().UnixNano()), nil
}

func normalizeSlug(input string) string {
	s := strings.TrimSpace(strings.ToLower(input))
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.Join(strings.Fields(s), "-")
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-")
}

func normalizeTags(input []string) model.StringList {
	if len(input) == 0 {
		return model.StringList{}
	}
	seen := make(map[string]struct{}, len(input))
	out := make([]string, 0, len(input))
	for _, raw := range input {
		t := strings.TrimSpace(strings.ToLower(raw))
		if t == "" {
			continue
		}
		t = strings.ReplaceAll(t, "，", ",")
		t = strings.Join(strings.Fields(t), " ")
		if len(t) > 32 {
			t = t[:32]
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
		if len(out) >= 10 {
			break
		}
	}
	return model.StringList(out)
}

func equalTags(a, b model.StringList) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func isSlugDuplicateErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate entry") && strings.Contains(msg, "slug")
}
