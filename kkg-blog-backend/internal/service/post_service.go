package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"awesomeProject/internal/model"
	"awesomeProject/internal/repository"
	"awesomeProject/internal/search"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
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

func (s *PostService) CreateDraft(authorID uint64, title string, slug string, summary string, tags []string, rawContent string, draftNote *string) (*model.Post, error) {
	title = strings.TrimSpace(title)
	baseSlug := normalizeSlug(slug)
	if title == "" {
		return nil, errors.New("title is required")
	}
	if baseSlug == "" {
		baseSlug = normalizeSlug(title)
	}
	if baseSlug == "" {
		baseSlug = fmt.Sprintf("post-%d", time.Now().Unix())
	}
	finalSlug, err := s.makeUniqueSlug(baseSlug, authorID)
	if err != nil {
		return nil, err
	}
	note := "初始草稿"
	if draftNote != nil {
		note = strings.TrimSpace(*draftNote)
	}
	post := &model.Post{
		AuthorID:         authorID,
		Version:          0,
		PublishedVersion: 0,
		Title:            title,
		Slug:             finalSlug,
		Summary:          strings.TrimSpace(summary),
		Tags:             normalizeTags(tags),
		RawContent:       rawContent,
		HTMLContent:      rawContent,
		Status:           "draft",
		Visibility:       "public",
	}

	err = s.posts.DB().Transaction(func(tx *gorm.DB) error {
		txRepo := repository.NewPostRepository(tx)
		if err := txRepo.Create(post); err != nil {
			return err
		}
		firstDraft := &model.PostVersion{
			PostID:      post.ID,
			Version:     1,
			DraftNote:   note,
			Title:       post.Title,
			Summary:     post.Summary,
			Tags:        post.Tags,
			RawContent:  post.RawContent,
			HTMLContent: post.HTMLContent,
			Status:      "draft",
			Visibility:  post.Visibility,
			OperatorID:  authorID,
		}
		return txRepo.CreateVersion(firstDraft)
	})
	if err != nil {
		if isSlugDuplicateErr(err) {
			return nil, errors.New("slug already exists for current user")
		}
		return nil, err
	}
	return post, nil
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

func (s *PostService) CreateDraftCopy(postID uint64, authorID uint64, fromVersion int, draftNote *string) (*model.PostVersion, error) {
	var out *model.PostVersion
	err := s.posts.DB().Transaction(func(tx *gorm.DB) error {
		txRepo := repository.NewPostRepository(tx)
		post, err := txRepo.GetByIDForAuthor(postID, authorID)
		if err != nil {
			return err
		}
		if post == nil {
			return errors.New("post not found")
		}

		latest, err := txRepo.GetLatestVersion(postID)
		if err != nil {
			return err
		}
		if latest == nil {
			return errors.New("version not found")
		}

		sourceVersion := fromVersion
		if sourceVersion <= 0 {
			if post.PublishedVersion > 0 {
				sourceVersion = post.PublishedVersion
			} else {
				sourceVersion = latest.Version
			}
		}

		source, err := txRepo.GetVersion(postID, sourceVersion)
		if err != nil {
			return err
		}
		if source == nil {
			return errors.New("version not found")
		}

		nextVersion := latest.Version + 1
		note := "副本自 v" + strconv.Itoa(sourceVersion)
		if draftNote != nil {
			note = strings.TrimSpace(*draftNote)
		}
		copyDraft := &model.PostVersion{
			PostID:      postID,
			Version:     nextVersion,
			DraftNote:   note,
			Title:       post.Title,
			Summary:     post.Summary,
			Tags:        post.Tags,
			RawContent:  source.RawContent,
			HTMLContent: source.HTMLContent,
			Status:      "draft",
			Visibility:  source.Visibility,
			OperatorID:  authorID,
		}
		if err := txRepo.CreateVersion(copyDraft); err != nil {
			return err
		}
		out = copyDraft
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PostService) SaveDraftByVersion(postID uint64, authorID uint64, version int, title string, summary string, rawContent string, draftNote *string) (*model.PostVersion, error) {
	if version <= 0 {
		return nil, errors.New("invalid version")
	}

	var out *model.PostVersion
	err := s.posts.DB().Transaction(func(tx *gorm.DB) error {
		txRepo := repository.NewPostRepository(tx)
		post, err := txRepo.GetByIDForAuthor(postID, authorID)
		if err != nil {
			return err
		}
		if post == nil {
			return errors.New("post not found")
		}

		draft, err := txRepo.GetVersion(postID, version)
		if err != nil {
			return err
		}
		if draft == nil {
			return errors.New("version not found")
		}
		isEditingPublished := post.PublishedVersion == version

		nextTitle := post.Title
		if strings.TrimSpace(title) != "" {
			nextTitle = strings.TrimSpace(title)
		}
		nextSummary := strings.TrimSpace(summary)
		nextRaw := rawContent
		nextDraftNote := draft.DraftNote
		if draftNote != nil {
			nextDraftNote = strings.TrimSpace(*draftNote)
		}
		if draft.Title == nextTitle &&
			draft.Summary == nextSummary &&
			equalTags(draft.Tags, post.Tags) &&
			draft.RawContent == nextRaw &&
			draft.DraftNote == nextDraftNote &&
			post.Title == nextTitle &&
			post.Summary == nextSummary &&
			(!isEditingPublished || (post.RawContent == nextRaw && post.HTMLContent == nextRaw)) {
			out = draft
			return nil
		}

		// 标题/摘要是文章级元信息，保存任意草稿时统一同步到全部版本。
		if post.Title != nextTitle || post.Summary != nextSummary || !equalTags(post.Tags, draft.Tags) {
			post.Title = nextTitle
			post.Summary = nextSummary
			post.Tags = normalizeTags(draft.Tags)
			if err := txRepo.Update(post); err != nil {
				return err
			}
			if err := txRepo.UpdateAllVersionMeta(postID, nextTitle, nextSummary, post.Tags, authorID); err != nil {
				return err
			}
		}

		updateFields := map[string]interface{}{
			"draft_note":   nextDraftNote,
			"raw_content":  nextRaw,
			"html_content": nextRaw,
			"operator_id":  authorID,
		}
		if isEditingPublished {
			updateFields["status"] = "published"
			updateFields["publish_at"] = post.PublishAt
		} else {
			updateFields["status"] = "draft"
			updateFields["publish_at"] = nil
		}
		if err := txRepo.UpdateDraftByVersion(postID, version, updateFields); err != nil {
			return err
		}
		if isEditingPublished {
			post.RawContent = nextRaw
			post.HTMLContent = nextRaw
			post.Status = "published"
			if err := txRepo.Update(post); err != nil {
				return err
			}
		}
		updated, err := txRepo.GetVersion(postID, version)
		if err != nil {
			return err
		}
		out = updated
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PostService) PublishDraft(postID uint64, authorID uint64, version int) (*model.Post, error) {
	if version <= 0 {
		return nil, errors.New("invalid version")
	}

	var out *model.Post
	err := s.posts.DB().Transaction(func(tx *gorm.DB) error {
		txRepo := repository.NewPostRepository(tx)
		post, err := txRepo.GetByIDForAuthor(postID, authorID)
		if err != nil {
			return err
		}
		if post == nil {
			return errors.New("post not found")
		}

		draft, err := txRepo.GetVersion(postID, version)
		if err != nil {
			return err
		}
		if draft == nil {
			return errors.New("version not found")
		}

		now := time.Now()
		if err := txRepo.SetAllDraftStatus(postID, "draft", nil); err != nil {
			return err
		}
		if err := txRepo.UpdateDraftByVersion(postID, version, map[string]interface{}{
			"status":     "published",
			"publish_at": &now,
		}); err != nil {
			return err
		}

		post.Title = draft.Title
		post.Summary = draft.Summary
		post.Tags = normalizeTags(draft.Tags)
		post.RawContent = draft.RawContent
		post.HTMLContent = draft.HTMLContent
		post.Status = "published"
		post.PublishAt = &now
		post.PublishedVersion = version
		post.Version++

		if err := txRepo.Update(post); err != nil {
			return err
		}
		out = post
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.indexPost(out)
	s.invalidatePostCaches(postID)
	return out, nil
}

func (s *PostService) Unpublish(postID uint64, authorID uint64) (*model.Post, error) {
	var out *model.Post
	err := s.posts.DB().Transaction(func(tx *gorm.DB) error {
		txRepo := repository.NewPostRepository(tx)
		post, err := txRepo.GetByIDForAuthor(postID, authorID)
		if err != nil {
			return err
		}
		if post == nil {
			return errors.New("post not found")
		}
		if post.PublishedVersion <= 0 || post.Status != "published" {
			return errors.New("post is not published")
		}

		if err := txRepo.UpdateDraftByVersion(postID, post.PublishedVersion, map[string]interface{}{
			"status":      "draft",
			"publish_at":  nil,
			"operator_id": authorID,
		}); err != nil {
			return err
		}

		post.Status = "draft"
		post.PublishAt = nil
		post.PublishedVersion = 0
		post.Version++
		if err := txRepo.Update(post); err != nil {
			return err
		}
		out = post
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.deletePostFromIndex(postID)
	s.invalidatePostCaches(postID)
	return out, nil
}

func (s *PostService) Get(postID uint64, actorUserID uint64) (*model.Post, error) {
	post, err := s.posts.GetByID(postID)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, errors.New("post not found")
	}

	if post.Status != "published" && post.AuthorID != actorUserID {
		return nil, errors.New("forbidden")
	}
	return post, nil
}

func (s *PostService) GetMine(postID uint64, authorID uint64) (*model.Post, error) {
	post, err := s.posts.GetByIDForAuthor(postID, authorID)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, errors.New("post not found")
	}
	return post, nil
}

func (s *PostService) GetDraft(postID uint64, authorID uint64, version int) (*model.PostVersion, error) {
	if version <= 0 {
		return nil, errors.New("invalid version")
	}
	post, err := s.posts.GetByIDForAuthor(postID, authorID)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, errors.New("post not found")
	}
	draft, err := s.posts.GetVersion(postID, version)
	if err != nil {
		return nil, err
	}
	if draft == nil {
		return nil, errors.New("version not found")
	}
	return draft, nil
}

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

func (s *PostService) ListDrafts(postID uint64, authorID uint64, limit int) ([]model.PostVersion, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	post, err := s.posts.GetByIDForAuthor(postID, authorID)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, errors.New("post not found")
	}
	return s.posts.ListVersions(postID, limit)
}

func (s *PostService) DeleteDraft(postID uint64, authorID uint64, version int) error {
	if version <= 0 {
		return errors.New("invalid version")
	}

	return s.posts.DB().Transaction(func(tx *gorm.DB) error {
		txRepo := repository.NewPostRepository(tx)
		post, err := txRepo.GetByIDForAuthor(postID, authorID)
		if err != nil {
			return err
		}
		if post == nil {
			return errors.New("post not found")
		}
		if post.PublishedVersion == version {
			return errors.New("cannot delete published draft")
		}

		target, err := txRepo.GetVersion(postID, version)
		if err != nil {
			return err
		}
		if target == nil {
			return errors.New("version not found")
		}

		count, err := txRepo.CountVersions(postID)
		if err != nil {
			return err
		}
		if count <= 1 {
			return errors.New("at least one draft must be kept")
		}

		return txRepo.DeleteVersion(postID, version)
	})
}

func (s *PostService) DeletePost(postID uint64, authorID uint64) error {
	err := s.posts.DB().Transaction(func(tx *gorm.DB) error {
		txRepo := repository.NewPostRepository(tx)
		post, err := txRepo.GetByIDForAuthor(postID, authorID)
		if err != nil {
			return err
		}
		if post == nil {
			return errors.New("post not found")
		}
		post.Status = "hidden"
		post.PublishAt = nil
		post.PublishedVersion = 0
		post.Version++
		return txRepo.Update(post)
	})
	if err != nil {
		return err
	}
	s.deletePostFromIndex(postID)
	s.invalidatePostCaches(postID)
	return nil
}

func (s *PostService) DeletePostAsManager(postID uint64) error {
	err := s.posts.DB().Transaction(func(tx *gorm.DB) error {
		txRepo := repository.NewPostRepository(tx)
		post, err := txRepo.GetByID(postID)
		if err != nil {
			return err
		}
		if post == nil {
			return errors.New("post not found")
		}
		post.Status = "hidden"
		post.PublishAt = nil
		post.PublishedVersion = 0
		post.Version++
		return txRepo.Update(post)
	})
	if err != nil {
		return err
	}
	s.deletePostFromIndex(postID)
	s.invalidatePostCaches(postID)
	return nil
}

func (s *PostService) UpdateMeta(postID uint64, authorID uint64, title string, summary string, tags []string) (*model.Post, error) {
	title = strings.TrimSpace(title)
	summary = strings.TrimSpace(summary)
	tags = normalizeTags(tags)
	if title == "" {
		return nil, errors.New("title is required")
	}

	var out *model.Post
	err := s.posts.DB().Transaction(func(tx *gorm.DB) error {
		txRepo := repository.NewPostRepository(tx)
		post, err := txRepo.GetByIDForAuthor(postID, authorID)
		if err != nil {
			return err
		}
		if post == nil {
			return errors.New("post not found")
		}
		if post.Title == title && post.Summary == summary && equalTags(post.Tags, tags) {
			out = post
			return nil
		}

		post.Title = title
		post.Summary = summary
		post.Tags = tags
		post.Version++
		if err := txRepo.Update(post); err != nil {
			return err
		}
		if err := txRepo.UpdateAllVersionMeta(postID, title, summary, tags, authorID); err != nil {
			return err
		}
		out = post
		return nil
	})
	if err != nil {
		return nil, err
	}
	if out != nil && out.Status == "published" {
		s.indexPost(out)
		s.invalidatePostCaches(postID)
	}
	return out, nil
}

// Legacy compatibility endpoints
func (s *PostService) SaveDraft(postID uint64, authorID uint64, title string, summary string, rawContent string) (*model.Post, error) {
	post, err := s.posts.GetByIDForAuthor(postID, authorID)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, errors.New("post not found")
	}
	target := post.PublishedVersion
	if target <= 0 {
		latest, err := s.posts.GetLatestVersion(postID)
		if err != nil {
			return nil, err
		}
		if latest == nil {
			return nil, errors.New("version not found")
		}
		target = latest.Version
	}
	if _, err := s.SaveDraftByVersion(postID, authorID, target, title, summary, rawContent, nil); err != nil {
		return nil, err
	}
	return s.posts.GetByID(postID)
}

func (s *PostService) Publish(postID uint64, authorID uint64) (*model.Post, error) {
	latest, err := s.posts.GetLatestVersion(postID)
	if err != nil {
		return nil, err
	}
	if latest == nil {
		return nil, errors.New("version not found")
	}
	return s.PublishDraft(postID, authorID, latest.Version)
}

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

func (s *PostService) UnpublishLegacy(postID uint64, authorID uint64) (*model.Post, error) {
	return s.Unpublish(postID, authorID)
}

func (s *PostService) ListVersions(postID uint64, authorID uint64, limit int) ([]model.PostVersion, error) {
	return s.ListDrafts(postID, authorID, limit)
}

func (s *PostService) Rollback(postID uint64, authorID uint64, version int) (*model.Post, error) {
	return s.PublishDraft(postID, authorID, version)
}

func (s *PostService) DeleteVersion(postID uint64, authorID uint64, version int) error {
	return s.DeleteDraft(postID, authorID, version)
}
