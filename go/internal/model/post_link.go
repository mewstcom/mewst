package model

import "time"

// PostLinkは投稿とリンクの関連付けのドメインモデル。リンクカードを投稿に
// 紐づける。
type PostLink struct {
	ID        PostLinkID
	PostID    PostID
	LinkID    LinkID
	CreatedAt time.Time
	UpdatedAt time.Time
}
