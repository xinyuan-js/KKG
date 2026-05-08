package repository

import (
	"errors"
	"strings"

	"awesomeProject/internal/model"

	"gorm.io/gorm"
)

type PostRepository struct {
	db *gorm.DB
}

const postAuthorSelect = "CASE WHEN users.status = -1 THEN '用户已删除或注销' ELSE users.username END AS author_name, CASE WHEN users.status = -1 THEN '' ELSE users.avatar_url END AS author_avatar_url"

type PostEngagement struct {
	LikeCount     int64 `json:"like_count"`
	FavoriteCount int64 `json:"favorite_count"`
	CommentCount  int64 `json:"comment_count"`
	Liked         bool  `json:"liked"`
	Favorited     bool  `json:"favorited"`
}

func NewPostRepository(db *gorm.DB) *PostRepository {
	return &PostRepository{db: db}
}

func (r *PostRepository) DB() *gorm.DB {
	return r.db
}

func (r *PostRepository) Create(post *model.Post) error {
	return r.db.Create(post).Error
}

func (r *PostRepository) ExistsSlug(slug string) (bool, error) {
	var count int64
	err := r.db.Unscoped().Model(&model.Post{}).Where("slug = ?", slug).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *PostRepository) ExistsSlugForAuthor(authorID uint64, slug string) (bool, error) {
	var count int64
	err := r.db.Unscoped().Model(&model.Post{}).Where("author_id = ? AND slug = ?", authorID, slug).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *PostRepository) GetByID(id uint64) (*model.Post, error) {
	var post model.Post
	err := r.db.
		Table("posts").
		Select("posts.*, "+postAuthorSelect).
		Joins("LEFT JOIN users ON users.id = posts.author_id").
		Where("posts.id = ?", id).
		First(&post).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &post, nil
}

func (r *PostRepository) GetByIDForAuthor(id uint64, authorID uint64) (*model.Post, error) {
	var post model.Post
	err := r.db.Where("id = ? AND author_id = ?", id, authorID).First(&post).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &post, nil
}

func (r *PostRepository) Update(post *model.Post) error {
	return r.db.Save(post).Error
}

func (r *PostRepository) ListPublished(limit int) ([]model.Post, error) {
	var posts []model.Post
	err := r.db.
		Table("posts").
		Select("posts.*, "+postAuthorSelect).
		Joins("LEFT JOIN users ON users.id = posts.author_id").
		Where("posts.status = ?", "published").
		Order("posts.publish_at DESC").
		Limit(limit).
		Find(&posts).Error
	return posts, err
}

func (r *PostRepository) ListByAuthor(authorID uint64, limit int) ([]model.Post, error) {
	var posts []model.Post
	err := r.db.Where("author_id = ?", authorID).Order("updated_at DESC").Limit(limit).Find(&posts).Error
	return posts, err
}

func (r *PostRepository) ListPublishedByAuthor(authorID uint64, limit int) ([]model.Post, error) {
	var posts []model.Post
	err := r.db.
		Table("posts").
		Select("posts.*, "+postAuthorSelect).
		Joins("LEFT JOIN users ON users.id = posts.author_id").
		Where("posts.author_id = ? AND posts.status = ?", authorID, "published").
		Order("posts.publish_at DESC").
		Limit(limit).
		Find(&posts).Error
	return posts, err
}

func (r *PostRepository) CountPublishedByAuthor(authorID uint64) (int64, error) {
	var count int64
	err := r.db.Model(&model.Post{}).
		Where("author_id = ? AND status = ?", authorID, "published").
		Count(&count).Error
	return count, err
}

func (r *PostRepository) ListPublishedWithCommentCount(limit int) ([]model.Post, error) {
	var posts []model.Post
	err := r.db.
		Table("posts").
		Select("posts.*, "+postAuthorSelect+", COUNT(comments.id) AS comment_count").
		Joins("LEFT JOIN users ON users.id = posts.author_id").
		Joins("LEFT JOIN comments ON comments.post_id = posts.id AND comments.status = ?", "normal").
		Where("posts.status = ?", "published").
		Group("posts.id").
		Order("posts.publish_at DESC").
		Limit(limit).
		Find(&posts).Error
	return posts, err
}

func (r *PostRepository) TopInteractedAuthorIDs(userID uint64, limit int) ([]uint64, error) {
	var authorIDs []uint64
	err := r.db.
		Table("comments").
		Select("posts.author_id").
		Joins("JOIN posts ON posts.id = comments.post_id").
		Where("comments.user_id = ? AND posts.status = ? AND posts.author_id <> ?", userID, "published", userID).
		Group("posts.author_id").
		Order("COUNT(comments.id) DESC").
		Limit(limit).
		Pluck("posts.author_id", &authorIDs).Error
	return authorIDs, err
}

func (r *PostRepository) ListPublishedByAuthors(authorIDs []uint64, limit int) ([]model.Post, error) {
	if len(authorIDs) == 0 {
		return []model.Post{}, nil
	}
	var posts []model.Post
	err := r.db.
		Table("posts").
		Select("posts.*, "+postAuthorSelect+", COUNT(comments.id) AS comment_count").
		Joins("LEFT JOIN users ON users.id = posts.author_id").
		Joins("LEFT JOIN comments ON comments.post_id = posts.id AND comments.status = ?", "normal").
		Where("posts.status = ? AND posts.author_id IN ?", "published", authorIDs).
		Group("posts.id").
		Order("posts.publish_at DESC").
		Limit(limit).
		Find(&posts).Error
	return posts, err
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

func (r *PostRepository) SearchPublishedByKeyword(keyword string, limit int) ([]model.Post, error) {
	var posts []model.Post
	kw := "%" + keyword + "%"
	err := r.db.
		Table("posts").
		Select("posts.*, "+postAuthorSelect).
		Joins("LEFT JOIN users ON users.id = posts.author_id").
		Where("posts.status = ? AND (posts.title LIKE ? OR posts.summary LIKE ?)", "published", kw, kw).
		Order("posts.publish_at DESC").
		Limit(limit).
		Find(&posts).Error
	return posts, err
}

func (r *PostRepository) SuggestPublishedTitles(keyword string, limit int) ([]string, error) {
	var titles []string
	kw := "%" + keyword + "%"
	err := r.db.Model(&model.Post{}).
		Where("status = ? AND title LIKE ?", "published", kw).
		Order("publish_at DESC").
		Limit(limit).
		Pluck("title", &titles).Error
	return titles, err
}

func (r *PostRepository) ListForAdmin(keyword string, status string, page int, pageSize int) ([]model.Post, int64, error) {
	q := r.db.Model(&model.Post{})
	kw := strings.TrimSpace(keyword)
	if kw != "" {
		like := "%" + kw + "%"
		q = q.Where("title LIKE ? OR summary LIKE ?", like, like)
	}
	sv := strings.TrimSpace(status)
	if sv != "" {
		q = q.Where("status = ?", sv)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var posts []model.Post
	err := r.db.
		Table("posts").
		Select("posts.*, "+postAuthorSelect).
		Joins("LEFT JOIN users ON users.id = posts.author_id").
		Where("posts.id IN (?)", q.Select("id")).
		Order("posts.updated_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&posts).Error
	return posts, total, err
}

func (r *PostRepository) CreateVersion(version *model.PostVersion) error {
	return r.db.Create(version).Error
}

func (r *PostRepository) ListVersions(postID uint64, limit int) ([]model.PostVersion, error) {
	var versions []model.PostVersion
	err := r.db.
		Where("post_id = ?", postID).
		Order("CASE WHEN status = 'published' THEN 0 ELSE 1 END ASC").
		Order("updated_at DESC").
		Order("version DESC").
		Limit(limit).
		Find(&versions).Error
	return versions, err
}

func (r *PostRepository) GetVersion(postID uint64, version int) (*model.PostVersion, error) {
	var data model.PostVersion
	err := r.db.Where("post_id = ? AND version = ?", postID, version).First(&data).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *PostRepository) DeleteVersion(postID uint64, version int) error {
	return r.db.Where("post_id = ? AND version = ?", postID, version).Delete(&model.PostVersion{}).Error
}

func (r *PostRepository) CountVersions(postID uint64) (int64, error) {
	var count int64
	err := r.db.Model(&model.PostVersion{}).Where("post_id = ?", postID).Count(&count).Error
	return count, err
}

func (r *PostRepository) GetLatestVersion(postID uint64) (*model.PostVersion, error) {
	var data model.PostVersion
	err := r.db.Where("post_id = ?", postID).Order("version DESC").First(&data).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *PostRepository) SetAllDraftStatus(postID uint64, status string, publishAt interface{}) error {
	return r.db.Model(&model.PostVersion{}).Where("post_id = ?", postID).Updates(map[string]interface{}{
		"status":     status,
		"publish_at": publishAt,
	}).Error
}

func (r *PostRepository) UpdateDraftByVersion(postID uint64, version int, fields map[string]interface{}) error {
	return r.db.Model(&model.PostVersion{}).Where("post_id = ? AND version = ?", postID, version).Updates(fields).Error
}

func (r *PostRepository) UpdateAllVersionMeta(postID uint64, title string, summary string, tags model.StringList, operatorID uint64) error {
	return r.db.Model(&model.PostVersion{}).
		Where("post_id = ?", postID).
		Updates(map[string]interface{}{
			"title":       title,
			"summary":     summary,
			"tags":        tags,
			"operator_id": operatorID,
		}).Error
}

func (r *PostRepository) DeleteVersionsByPostID(postID uint64) error {
	return r.db.Where("post_id = ?", postID).Delete(&model.PostVersion{}).Error
}

func (r *PostRepository) DeletePost(post *model.Post) error {
	return r.db.Delete(post).Error
}
