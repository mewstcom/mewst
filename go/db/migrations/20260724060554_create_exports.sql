-- migrate:up

-- Add a composite unique key on actors so exports can reference the
-- (actor_id, profile_id) pair. PostgreSQL requires the referenced columns of a
-- composite foreign key to be covered by a unique constraint; the primary key
-- on id alone is not enough. This lets exports reject rows whose acting actor
-- belongs to a different profile than the export target.
--
-- [Ja] exports が (actor_id, profile_id) の組を参照できるよう、actors に複合
-- ユニークキーを追加する。PostgreSQL は複合外部キーの参照先カラムがユニーク
-- 制約でカバーされていることを要求し、id 単独の主キーでは足りない。これにより
-- exports は、申請 actor がエクスポート対象と別のプロフィールに属する行を拒否
-- できる。
ALTER TABLE actors
    ADD CONSTRAINT actors_id_profile_id_key UNIQUE (id, profile_id);

CREATE TABLE exports (
    id uuid DEFAULT public.generate_ulid() NOT NULL PRIMARY KEY,
    -- Keep profile_id (export target) and actor_id (requester) separate: the
    -- retention scope follows the profile, while the requester resolves the
    -- notification recipient and audit trail.
    --
    -- [Ja] profile_id (エクスポート対象) と actor_id (申請者) を分けて保持する。
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
    -- Reject rows whose acting actor belongs to a different profile than the
    -- export target. ON DELETE NO ACTION (not CASCADE) because a succeeded
    -- export owns an R2 object that application code must delete first; per ADR
    -- 0002 the FK is an explicit safety net against orphaning that object.
    --
    -- [Ja] 申請 actor がエクスポート対象と別のプロフィールに属する行を拒否する。
    -- succeeded の行は先にアプリケーションが削除すべき R2 オブジェクトを持つため、
    -- CASCADE ではなく ON DELETE NO ACTION とする。ADR 0002 に従い、この FK は
    -- オブジェクトを孤児化させないための明示的な安全網。
    CONSTRAINT exports_actor_profile_fkey
        FOREIGN KEY (actor_id, profile_id)
        REFERENCES actors (id, profile_id) ON DELETE NO ACTION,
    CONSTRAINT exports_status_check
        CHECK (status IN ('queued', 'started', 'succeeded', 'failed')),
    CONSTRAINT exports_attempt_count_check
        CHECK (attempt_count >= 0),
    -- Pin the valid combinations of status and the state timestamps / object_key
    -- so an application bug cannot persist a contradictory row (e.g. succeeded
    -- without an object_key, or queued with a started_at).
    --
    -- [Ja] status と状態タイムスタンプ / object_key の妥当な組み合わせを固定し、
    -- アプリケーションのバグで矛盾した行 (例: object_key の無い succeeded、
    -- started_at のある queued) を永続化できないようにする。
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

-- Child-side index for resolving the requester and for the foreign-key check
-- when an actor is deleted, avoiding a full scan of exports.
--
-- [Ja] 申請者の解決と、actor 削除時の外部キー検査のための子側インデックス。
-- exports の全表走査を避ける。
CREATE INDEX index_exports_on_actor_id_and_profile_id
    ON exports (actor_id, profile_id);

-- Last line of defense against concurrent requests creating more than one
-- in-progress export per profile: a partial unique index over the active
-- statuses. succeeded / failed rows are excluded so they never block a new run.
--
-- [Ja] 同時リクエストがプロフィールごとに 2 件以上の進行中エクスポートを作るのを
-- 防ぐ最終防衛線として、active な status に対する部分ユニークインデックスを張る。
-- succeeded / failed の行は除外し、新規実行を妨げないようにする。
CREATE UNIQUE INDEX index_exports_on_active_profile_id
    ON exports (profile_id)
    WHERE status IN ('queued', 'started');

-- migrate:down

DROP TABLE IF EXISTS exports;
ALTER TABLE actors DROP CONSTRAINT IF EXISTS actors_id_profile_id_key;
