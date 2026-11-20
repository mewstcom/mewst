package model

import "time"

// HomeTimelinePostはプロフィールのホームタイムライン上のエントリのドメインモデル。
// 投稿をプロフィールのタイムラインに紐づけ、投稿のpublished_atを複製して持つことで、
// postsへjoinし直さずにタイムラインを並べ替えられるようにしている。
type HomeTimelinePost struct {
	ID          HomeTimelinePostID
	ProfileID   ProfileID
	PostID      PostID
	PublishedAt time.Time
}
