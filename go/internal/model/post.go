package model

import "time"

// Post is the domain model for a post.
// [Ja] Post は投稿のドメインモデル。
type Post struct {
	ID                 PostID
	ProfileID          ProfileID
	Content            string
	PublishedAt        time.Time
	OauthApplicationID OauthApplicationID
	DiscardedAt        *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
