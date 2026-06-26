package service

import (
	"awesomeProject/internal/model"
	"awesomeProject/internal/repository"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"strconv"
	"strings"
	"time"
)

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
