package service

import (
	"awesomeProject/internal/model"
	"errors"
)

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
