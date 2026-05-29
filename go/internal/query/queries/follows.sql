-- name: ListFollowsByTargetProfileID :many
-- Lists the follows whose target is the given profile. Their source profiles
-- are that profile's followers, which fanout uses to enqueue timeline delivery.
--
-- [Ja] target が指定プロフィールである follow を列挙する。その source プロフィールが
-- 当該プロフィールのフォロワーであり、fanout がタイムライン配信を enqueue する際に使う。
SELECT * FROM follows WHERE target_profile_id = $1;
