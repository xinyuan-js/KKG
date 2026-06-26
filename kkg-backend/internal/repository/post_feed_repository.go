package repository

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
