package repository

import (
	"awesomeProject/internal/model"
)

type PostEngagement struct {
	LikeCount     int64 `json:"like_count"`
	FavoriteCount int64 `json:"favorite_count"`
	CommentCount  int64 `json:"comment_count"`
	Liked         bool  `json:"liked"`
	Favorited     bool  `json:"favorited"`
}

func (r *PostRepository) ListFavoritesByUser(userID uint64, limit int) ([]model.Post, error) {
	var posts []model.Post
	err := r.db.
		Unscoped().
		Table("post_favorites pf").
		Select("posts.*, "+postAuthorSelect+", COUNT(comments.id) AS comment_count").
		Joins("JOIN posts ON posts.id = pf.post_id").
		Joins("LEFT JOIN users ON users.id = posts.author_id").
		Joins("LEFT JOIN comments ON comments.post_id = posts.id AND comments.status = ?", "normal").
		Where("pf.user_id = ? AND posts.status = ? AND posts.deleted_at IS NULL", userID, "published").
		Group("posts.id, pf.created_at").
		Order("pf.created_at DESC").
		Limit(limit).
		Find(&posts).Error
	return posts, err
}

func (r *PostRepository) IsLiked(postID uint64, userID uint64) (bool, error) {
	var count int64
	if err := r.db.Model(&model.PostLike{}).Where("post_id = ? AND user_id = ?", postID, userID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *PostRepository) IsFavorited(postID uint64, userID uint64) (bool, error) {
	var count int64
	if err := r.db.Model(&model.PostFavorite{}).Where("post_id = ? AND user_id = ?", postID, userID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *PostRepository) CreateLike(postID uint64, userID uint64) error {
	return r.db.Create(&model.PostLike{PostID: postID, UserID: userID}).Error
}

func (r *PostRepository) DeleteLike(postID uint64, userID uint64) error {
	return r.db.Where("post_id = ? AND user_id = ?", postID, userID).Delete(&model.PostLike{}).Error
}

func (r *PostRepository) CreateFavorite(postID uint64, userID uint64) error {
	return r.db.Create(&model.PostFavorite{PostID: postID, UserID: userID}).Error
}

func (r *PostRepository) DeleteFavorite(postID uint64, userID uint64) error {
	return r.db.Where("post_id = ? AND user_id = ?", postID, userID).Delete(&model.PostFavorite{}).Error
}

func (r *PostRepository) GetEngagement(postID uint64, userID uint64) (*PostEngagement, error) {
	var likeCount int64
	var favoriteCount int64
	var commentCount int64
	if err := r.db.Model(&model.PostLike{}).Where("post_id = ?", postID).Count(&likeCount).Error; err != nil {
		return nil, err
	}
	if err := r.db.Model(&model.PostFavorite{}).Where("post_id = ?", postID).Count(&favoriteCount).Error; err != nil {
		return nil, err
	}
	if err := r.db.Model(&model.Comment{}).Where("post_id = ? AND status = ?", postID, "normal").Count(&commentCount).Error; err != nil {
		return nil, err
	}
	liked := false
	favorited := false
	if userID > 0 {
		var mineLike int64
		var mineFav int64
		if err := r.db.Model(&model.PostLike{}).Where("post_id = ? AND user_id = ?", postID, userID).Count(&mineLike).Error; err != nil {
			return nil, err
		}
		if err := r.db.Model(&model.PostFavorite{}).Where("post_id = ? AND user_id = ?", postID, userID).Count(&mineFav).Error; err != nil {
			return nil, err
		}
		liked = mineLike > 0
		favorited = mineFav > 0
	}
	return &PostEngagement{
		LikeCount:     likeCount,
		FavoriteCount: favoriteCount,
		CommentCount:  commentCount,
		Liked:         liked,
		Favorited:     favorited,
	}, nil
}
