-- name: IsFeatureFlagEnabledForActor :one
-- 指定actorに対してフラグが有効かを返す。設定メニューなどのアプリ内制御で使う。
SELECT EXISTS(
    SELECT 1 FROM feature_flags
    WHERE actor_id = $1 AND name = $2
);

-- name: IsFeatureFlagEnabledForDevice :one
-- device_tokenまたはセッショントークン経由のactor_idでフラグが有効かを1クエリで判定する。
SELECT EXISTS(
    SELECT 1 FROM feature_flags ff
    WHERE ff.name = $3
    AND (
        (ff.device_token IS NOT NULL AND ff.device_token = $1)
        OR (ff.actor_id IS NOT NULL AND ff.actor_id = (
            SELECT s.actor_id FROM sessions s WHERE s.token = $2
        ))
    )
);
