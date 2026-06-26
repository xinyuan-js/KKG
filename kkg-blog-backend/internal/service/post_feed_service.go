package service

import (
	"awesomeProject/internal/model"
	"errors"
	"math"
	"sort"
	"time"
)

func (s *PostService) ListPublished(limit int) ([]model.Post, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.posts.ListPublished(limit)
}

func (s *PostService) ListFeed(feedType string, actorID uint64, limit int) ([]model.Post, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	switch feedType {
	case "latest":
		return s.posts.ListPublished(limit)
	case "hot":
		return s.listHotFeed(limit)
	case "recommend":
		return s.listRecommendFeed(actorID, limit)
	default:
		return nil, errors.New("invalid feed type")
	}
}

func (s *PostService) PageFeed(feedType string, actorID uint64, page int, pageSize int) ([]model.Post, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 50 {
		pageSize = 20
	}
	switch feedType {
	case "latest":
		return s.posts.PagePublished(page, pageSize)
	case "hot":
		return s.pageSortedFeed(page, pageSize, func(limit int) ([]model.Post, error) {
			return s.listHotFeed(limit)
		})
	case "recommend":
		return s.pageSortedFeed(page, pageSize, func(limit int) ([]model.Post, error) {
			return s.listRecommendFeed(actorID, limit)
		})
	default:
		return nil, 0, errors.New("invalid feed type")
	}
}

func (s *PostService) pageSortedFeed(page int, pageSize int, load func(limit int) ([]model.Post, error)) ([]model.Post, int64, error) {
	candidates, err := load(300)
	if err != nil {
		return nil, 0, err
	}
	total := int64(len(candidates))
	start := (page - 1) * pageSize
	if start >= len(candidates) {
		return []model.Post{}, total, nil
	}
	end := start + pageSize
	if end > len(candidates) {
		end = len(candidates)
	}
	return candidates[start:end], total, nil
}

func (s *PostService) ListRanking(limit int) ([]model.Post, error) {
	return s.ListRankingByPeriod(limit, "all")
}

func (s *PostService) ListRankingByPeriod(limit int, period string) ([]model.Post, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if cached := s.getRankingCache(period, limit); cached != nil {
		return cached, nil
	}
	candidates, err := s.posts.ListPublishedWithCommentCount(300)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	var cutoff time.Time
	switch period {
	case "24h":
		cutoff = now.Add(-24 * time.Hour)
	case "7d":
		cutoff = now.Add(-7 * 24 * time.Hour)
	case "30d":
		cutoff = now.Add(-30 * 24 * time.Hour)
	case "all", "":
	default:
		return nil, errors.New("invalid ranking period")
	}
	filtered := make([]model.Post, 0, len(candidates))
	for _, p := range candidates {
		if cutoff.IsZero() || (p.PublishAt != nil && p.PublishAt.After(cutoff)) {
			filtered = append(filtered, p)
		}
	}
	candidates = filtered

	for i := range candidates {
		eng, err := s.posts.GetEngagement(candidates[i].ID, 0)
		if err != nil {
			return nil, err
		}
		hours := 1.0
		if candidates[i].PublishAt != nil {
			hours = math.Max(1, now.Sub(*candidates[i].PublishAt).Hours())
		}
		candidates[i].FeedScore = (float64(eng.LikeCount)*1.5 + float64(eng.FavoriteCount)*2.5 + float64(eng.CommentCount)*2 + 1) / math.Pow(hours+2, 1.2)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].FeedScore > candidates[j].FeedScore
	})
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	s.setRankingCache(period, limit, candidates)
	return candidates, nil
}

func (s *PostService) listHotFeed(limit int) ([]model.Post, error) {
	candidates, err := s.posts.ListPublishedWithCommentCount(300)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	for i := range candidates {
		hours := 1.0
		if candidates[i].PublishAt != nil {
			hours = math.Max(1, now.Sub(*candidates[i].PublishAt).Hours())
		}
		// 主流热度做法：互动量 + 时间衰减。
		candidates[i].FeedScore = (float64(candidates[i].CommentCount)*2 + 1) / math.Pow(hours+2, 1.3)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].FeedScore == candidates[j].FeedScore {
			if candidates[i].PublishAt == nil || candidates[j].PublishAt == nil {
				return candidates[i].UpdatedAt.After(candidates[j].UpdatedAt)
			}
			return candidates[i].PublishAt.After(*candidates[j].PublishAt)
		}
		return candidates[i].FeedScore > candidates[j].FeedScore
	})
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates, nil
}

func (s *PostService) listRecommendFeed(actorID uint64, limit int) ([]model.Post, error) {
	// 未登录：推荐流降级为热门流。
	if actorID == 0 {
		return s.listHotFeed(limit)
	}
	authorIDs, err := s.posts.TopInteractedAuthorIDs(actorID, 12)
	if err != nil {
		return nil, err
	}
	primary, err := s.posts.ListPublishedByAuthors(authorIDs, limit)
	if err != nil {
		return nil, err
	}
	if len(primary) >= limit {
		return primary[:limit], nil
	}

	hotFallback, err := s.listHotFeed(limit * 2)
	if err != nil {
		return nil, err
	}
	exists := make(map[uint64]struct{}, len(primary))
	for _, p := range primary {
		exists[p.ID] = struct{}{}
	}
	for _, p := range hotFallback {
		if _, ok := exists[p.ID]; ok {
			continue
		}
		if p.AuthorID == actorID {
			continue
		}
		primary = append(primary, p)
		exists[p.ID] = struct{}{}
		if len(primary) >= limit {
			break
		}
	}
	return primary, nil
}
