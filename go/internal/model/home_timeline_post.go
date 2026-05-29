package model

import "time"

// HomeTimelinePost is the domain model for an entry on a profile's home timeline.
// It links a post into a profile's timeline with the post's published_at copied
// over, so the timeline can be ordered without joining back to posts.
//
// [Ja] HomeTimelinePost はプロフィールのホームタイムライン上のエントリのドメインモデル。
// 投稿をプロフィールのタイムラインに紐づけ、投稿の published_at を複製して持つことで、
// posts へ join し直さずにタイムラインを並べ替えられるようにしている。
type HomeTimelinePost struct {
	ID          HomeTimelinePostID
	ProfileID   ProfileID
	PostID      PostID
	PublishedAt time.Time
}
