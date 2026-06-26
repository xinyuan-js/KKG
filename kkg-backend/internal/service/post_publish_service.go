package service

import (
	"kkg-backend/internal/model"
	"kkg-backend/internal/repository"
	"errors"
	"gorm.io/gorm"
	"time"
)

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
