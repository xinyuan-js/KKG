package service

import (
	"errors"
	"strings"

	"awesomeProject/internal/model"
	"awesomeProject/internal/repository"
)

type CommentService struct {
	comments      *repository.CommentRepository
	posts         *repository.PostRepository
	notifications *repository.NotificationRepository
}

func NewCommentService(
	comments *repository.CommentRepository,
	posts *repository.PostRepository,
	notifications *repository.NotificationRepository,
) *CommentService {
	return &CommentService{comments: comments, posts: posts, notifications: notifications}
}

func (s *CommentService) ListByPost(postID uint64, limit int) ([]model.Comment, error) {
	if postID == 0 {
		return nil, errors.New("invalid post id")
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	post, err := s.posts.GetByID(postID)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, errors.New("post not found")
	}
	if post.Status != "published" {
		return nil, errors.New("forbidden")
	}
	return s.comments.ListByPostID(postID, limit)
}

func (s *CommentService) Create(postID uint64, userID uint64, parentID *uint64, content string) (*model.Comment, error) {
	if postID == 0 {
		return nil, errors.New("invalid post id")
	}
	if userID == 0 {
		return nil, errors.New("invalid user")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("content is required")
	}
	if len([]rune(content)) > 2000 {
		return nil, errors.New("content too long")
	}

	post, err := s.posts.GetByID(postID)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, errors.New("post not found")
	}
	if post.Status != "published" {
		return nil, errors.New("cannot comment unpublished post")
	}

	var parent *model.Comment
	if parentID != nil && *parentID > 0 {
		parent, err = s.comments.GetByID(*parentID)
		if err != nil {
			return nil, err
		}
		if parent == nil || parent.PostID != postID {
			return nil, errors.New("parent comment not found")
		}
	}

	comment := &model.Comment{
		PostID:   postID,
		UserID:   userID,
		ParentID: parentID,
		Content:  content,
		Status:   "normal",
	}
	if err := s.comments.Create(comment); err != nil {
		return nil, err
	}
	created, err := s.comments.GetByID(comment.ID)
	if err != nil {
		return nil, err
	}
	if created == nil {
		return nil, errors.New("comment not found")
	}

	if parent != nil && parent.UserID != userID && s.notifications != nil {
		_ = s.notifications.Create(&model.Notification{
			ReceiverID:      parent.UserID,
			ActorID:         userID,
			PostID:          postID,
			CommentID:       created.ID,
			ParentCommentID: &parent.ID,
			Type:            "reply_comment",
			IsRead:          false,
		})
	}

	return created, nil
}
