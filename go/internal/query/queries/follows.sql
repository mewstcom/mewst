-- name: ListFollowsByTargetProfileID :many
-- targetが指定プロフィールであるfollowを列挙する。そのsourceプロフィールが
-- 当該プロフィールのフォロワーであり、fanoutがタイムライン配信をenqueueする際に使う。
SELECT * FROM follows WHERE target_profile_id = $1;
