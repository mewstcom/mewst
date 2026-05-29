-- name: CreateHomeTimelinePost :one
-- Idempotently adds a post to a profile's home timeline. On conflict with the
-- unique (profile_id, post_id) index it leaves the existing row untouched
-- (the no-op DO UPDATE preserves the original published_at) so RETURNING still
-- yields the row, mirroring Rails' home_timeline.add_post! (first_or_create!).
--
-- [Ja] 投稿をプロフィールのホームタイムラインに冪等に追加する。unique な
-- (profile_id, post_id) インデックスで衝突した場合は既存行をそのまま残し
-- (no-op の DO UPDATE で元の published_at を保持)、RETURNING で行を返せるように
-- する。Rails の home_timeline.add_post! (first_or_create!) を踏襲している。
INSERT INTO home_timeline_posts (profile_id, post_id, published_at)
VALUES ($1, $2, $3)
ON CONFLICT (profile_id, post_id)
DO UPDATE SET published_at = home_timeline_posts.published_at
RETURNING *;
