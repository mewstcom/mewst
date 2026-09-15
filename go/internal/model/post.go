package model

import "time"

// MaximumPostContentLengthは投稿本文に許可される最大文字数。Railsの
// PostRecord::MAXIMUM_CONTENT_LENGTHに揃え、どちら側で作成した投稿も同じ上限になる
// ようにしている。validatorと投稿フォームの文字数カウンターの両方から参照される。
const MaximumPostContentLength = 160

// Postは投稿のドメインモデル。
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
