package service

import (
	"awesomeProject/internal/model"
	"awesomeProject/internal/repository"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

func (s *PostService) ToggleLike(postID uint64, userID uint64) (*repository.PostEngagement, error) {
	if userID == 0 {
		return nil, errors.New("invalid user context")
	}
	post, err := s.posts.GetByID(postID)
	if err != nil {
		return nil, err
	}
	if post == nil || post.Status != "published" {
		return nil, errors.New("post not found")
	}
	liked, err := s.posts.IsLiked(postID, userID)
	if err != nil {
		return nil, err
	}
	if liked {
		if err := s.posts.DeleteLike(postID, userID); err != nil {
			return nil, err
		}
	} else {
		if err := s.posts.CreateLike(postID, userID); err != nil {
			return nil, err
		}
	}
	s.invalidatePostCaches(postID)
	return s.posts.GetEngagement(postID, userID)
}

func (s *PostService) ToggleFavorite(postID uint64, userID uint64) (*repository.PostEngagement, error) {
	if userID == 0 {
		return nil, errors.New("invalid user context")
	}
	post, err := s.posts.GetByID(postID)
	if err != nil {
		return nil, err
	}
	if post == nil || post.Status != "published" {
		return nil, errors.New("post not found")
	}
	favorited, err := s.posts.IsFavorited(postID, userID)
	if err != nil {
		return nil, err
	}
	if favorited {
		if err := s.posts.DeleteFavorite(postID, userID); err != nil {
			return nil, err
		}
	} else {
		if err := s.posts.CreateFavorite(postID, userID); err != nil {
			return nil, err
		}
	}
	s.invalidatePostCaches(postID)
	return s.posts.GetEngagement(postID, userID)
}

func (s *PostService) GetEngagement(postID uint64, userID uint64) (*repository.PostEngagement, error) {
	post, err := s.posts.GetByID(postID)
	if err != nil {
		return nil, err
	}
	if post == nil || post.Status != "published" {
		return nil, errors.New("post not found")
	}
	if userID == 0 {
		if cached := s.getEngagementCache(postID); cached != nil {
			return cached, nil
		}
	}
	eng, err := s.posts.GetEngagement(postID, userID)
	if err != nil {
		return nil, err
	}
	if userID == 0 {
		s.setEngagementCache(postID, eng)
	}
	return eng, nil
}

func (s *PostService) ListMyFavorites(userID uint64, limit int) ([]model.Post, error) {
	if userID == 0 {
		return nil, errors.New("invalid user context")
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.posts.ListFavoritesByUser(userID, limit)
}

func (s *PostService) engagementCacheKey(postID uint64) string {
	return fmt.Sprintf("eng:post:%d", postID)
}

func (s *PostService) rankingCacheKey(period string, limit int) string {
	return fmt.Sprintf("rank:posts:%s:%d", period, limit)
}

func (s *PostService) getEngagementCache(postID uint64) *repository.PostEngagement {
	if s.redis == nil {
		return nil
	}
	raw, err := s.redis.Get(s.redCtx, s.engagementCacheKey(postID)).Result()
	if err != nil {
		return nil
	}
	var data repository.PostEngagement
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return nil
	}
	return &data
}

func (s *PostService) setEngagementCache(postID uint64, data *repository.PostEngagement) {
	if s.redis == nil || data == nil {
		return
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	_ = s.redis.Set(s.redCtx, s.engagementCacheKey(postID), raw, 20*time.Second).Err()
}

func (s *PostService) getRankingCache(period string, limit int) []model.Post {
	if s.redis == nil {
		return nil
	}
	raw, err := s.redis.Get(s.redCtx, s.rankingCacheKey(period, limit)).Result()
	if err != nil {
		return nil
	}
	var data []model.Post
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return nil
	}
	return data
}

func (s *PostService) setRankingCache(period string, limit int, data []model.Post) {
	if s.redis == nil || len(data) == 0 {
		return
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	_ = s.redis.Set(s.redCtx, s.rankingCacheKey(period, limit), raw, 60*time.Second).Err()
}

func (s *PostService) invalidatePostCaches(postID uint64) {
	if s.redis == nil {
		return
	}
	_ = s.redis.Del(s.redCtx, s.engagementCacheKey(postID)).Err()
	iter := s.redis.Scan(s.redCtx, 0, "rank:posts:*", 200).Iterator()
	keys := make([]string, 0, 16)
	for iter.Next(s.redCtx) {
		keys = append(keys, iter.Val())
	}
	if len(keys) > 0 {
		_ = s.redis.Del(s.redCtx, keys...).Err()
	}
}
