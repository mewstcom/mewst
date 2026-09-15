-- name: CreateHomeTimelinePost :one
-- 投稿をプロフィールのホームタイムラインに冪等に追加する。uniqueな
-- (profile_id, post_id) インデックスで衝突した場合は既存行をそのまま残し
-- (no-opのDO UPDATEで元のpublished_atを保持)、RETURNINGで行を返せるように
-- する。Railsのhome_timeline.add_post! (first_or_create!) を踏襲している。
INSERT INTO home_timeline_posts (profile_id, post_id, published_at)
VALUES ($1, $2, $3)
ON CONFLICT (profile_id, post_id)
DO UPDATE SET published_at = home_timeline_posts.published_at
RETURNING *;
