package service

import (
	"awesomeProject/internal/repository"
	"errors"
	"gorm.io/gorm"
)

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
