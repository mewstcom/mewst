-- migrate:up

-- exportsが (actor_id, profile_id) の組を参照できるよう、actorsに複合
-- ユニークキーを追加する。PostgreSQLは複合外部キーの参照先カラムがユニーク
-- 制約でカバーされていることを要求し、id単独の主キーでは足りない。これにより
-- exportsは、申請actorがエクスポート対象と別のプロフィールに属する行を拒否
-- できる。
ALTER TABLE actors
    ADD CONSTRAINT actors_id_profile_id_key UNIQUE (id, profile_id);

CREATE TABLE exports (
    id uuid DEFAULT public.generate_ulid() NOT NULL PRIMARY KEY,
    -- profile_id (エクスポート対象) とactor_id (申請者) を分けて保持する。
    -- 保持ポリシーのスコープはプロフィールに従い、申請者は通知先と監査の解決に
    -- 使う。
    profile_id uuid NOT NULL
        REFERENCES profiles (id) ON DELETE NO ACTION,
    actor_id uuid NOT NULL,
    status VARCHAR NOT NULL DEFAULT 'queued',
    object_key VARCHAR,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMP WITH TIME ZONE,
    finished_at TIMESTAMP WITH TIME ZONE,
    completion_notified_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    -- 申請actorがエクスポート対象と別のプロフィールに属する行を拒否する。
    -- succeededの行は先にアプリケーションが削除すべきR2オブジェクトを持つため、
    -- CASCADEではなくON DELETE NO ACTIONとする。ADR 0002に従い、このFKは
    -- オブジェクトを孤児化させないための明示的な安全網。
    CONSTRAINT exports_actor_profile_fkey
        FOREIGN KEY (actor_id, profile_id)
        REFERENCES actors (id, profile_id) ON DELETE NO ACTION,
    CONSTRAINT exports_status_check
        CHECK (status IN ('queued', 'started', 'succeeded', 'failed')),
    CONSTRAINT exports_attempt_count_check
        CHECK (attempt_count >= 0),
    -- statusと状態タイムスタンプ / object_keyの妥当な組み合わせを固定し、
    -- アプリケーションのバグで矛盾した行 (例: object_keyの無いsucceeded、
    -- started_atのあるqueued) を永続化できないようにする。
    CONSTRAINT exports_state_fields_check
        CHECK (
            (
                status = 'queued'
                AND object_key IS NULL
                AND started_at IS NULL
                AND finished_at IS NULL
            )
            OR (
                status = 'started'
                AND object_key IS NULL
                AND started_at IS NOT NULL
                AND finished_at IS NULL
            )
            OR (
                status = 'succeeded'
                AND object_key IS NOT NULL
                AND started_at IS NOT NULL
                AND finished_at IS NOT NULL
            )
            OR (
                status = 'failed'
                AND object_key IS NULL
                AND started_at IS NOT NULL
                AND finished_at IS NOT NULL
            )
        ),
    CONSTRAINT exports_completion_notified_at_check
        CHECK (completion_notified_at IS NULL OR status = 'succeeded')
);

CREATE INDEX index_exports_on_profile_id_and_created_at
    ON exports (profile_id, created_at DESC, id DESC);

-- 申請者の解決と、actor削除時の外部キー検査のための子側インデックス。
-- exportsの全表走査を避ける。
CREATE INDEX index_exports_on_actor_id_and_profile_id
    ON exports (actor_id, profile_id);

-- 同時リクエストがプロフィールごとに2件以上の進行中エクスポートを作るのを
-- 防ぐ最終防衛線として、activeなstatusに対する部分ユニークインデックスを張る。
-- succeeded / failedの行は除外し、新規実行を妨げないようにする。
CREATE UNIQUE INDEX index_exports_on_active_profile_id
    ON exports (profile_id)
    WHERE status IN ('queued', 'started');

-- migrate:down

DROP TABLE IF EXISTS exports;
ALTER TABLE actors DROP CONSTRAINT IF EXISTS actors_id_profile_id_key;
