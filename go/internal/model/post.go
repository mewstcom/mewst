package model

import "time"

// MaximumPostContentLength is the maximum number of characters allowed in a
// post body. It mirrors the Rails PostRecord::MAXIMUM_CONTENT_LENGTH so that
// posts created on either side share the same limit. Both the validator and
// the character counter on the post form reference this constant.
//
// [Ja] MaximumPostContentLength は投稿本文に許可される最大文字数。Rails の
// PostRecord::MAXIMUM_CONTENT_LENGTH に揃え、どちら側で作成した投稿も同じ上限になる
// ようにしている。validator と投稿フォームの文字数カウンターの両方から参照される。
const MaximumPostContentLength = 160

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
