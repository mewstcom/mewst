package model

import "time"

// PostLink is the domain model for the association between a post and a link.
// It attaches a link card to a post.
//
// [Ja] PostLink は投稿とリンクの関連付けのドメインモデル。リンクカードを投稿に
// 紐づける。
type PostLink struct {
	ID        PostLinkID
	PostID    PostID
	LinkID    LinkID
	CreatedAt time.Time
	UpdatedAt time.Time
}
