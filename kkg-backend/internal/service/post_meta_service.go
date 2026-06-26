package service

import (
	"kkg-backend/internal/model"
	"kkg-backend/internal/repository"
	"errors"
	"gorm.io/gorm"
	"strings"
)

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
