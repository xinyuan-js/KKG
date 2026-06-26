package service

import (
	"awesomeProject/internal/model"
	"errors"
)

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
