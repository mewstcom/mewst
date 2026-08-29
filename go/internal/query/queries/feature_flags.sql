-- name: IsFeatureFlagEnabledForActor :one
-- Reports whether the flag is enabled for the given actor, used for in-app control such as the settings menu.
-- [Ja] 指定 actor に対してフラグが有効かを返す。設定メニューなどのアプリ内制御で使う。
SELECT EXISTS(
    SELECT 1 FROM feature_flags
    WHERE actor_id = $1 AND name = $2
);

-- name: IsFeatureFlagEnabledForDevice :one
-- Reports whether the flag is enabled via device_token or the actor_id resolved from a session token, in a single query.
-- [Ja] device_token またはセッショントークン経由の actor_id でフラグが有効かを 1 クエリで判定する。
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
