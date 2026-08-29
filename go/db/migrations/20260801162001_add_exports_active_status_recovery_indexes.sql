-- migrate:up

-- Indexes for the reconciliation queries that walk stale queued and started
-- exports. The partial unique index on active rows already restricts the scan
-- to in-progress exports, but it is keyed by profile_id, so the ordering column
-- is missing and every page has to read all active rows and sort them. Keying
-- these on the ordering column lets the keyset cursor become the starting point
-- of the index scan, which is what bounds the work per page once a backlog has
-- built up.
--
-- [Ja] 滞留した queued / started のエクスポートを走査するリコンシリエーションの
-- クエリ用インデックス。active な行に対する部分ユニークインデックスは走査を進行中の
-- エクスポートに限定するが、キーが profile_id のため並び順のカラムを持たず、どの
-- ページでも active な行を全件読んでソートすることになる。並び順のカラムをキーに
-- することで keyset cursor が索引スキャンの開始位置になり、バックログが積み上がった
-- ときに 1 ページあたりの仕事量が有界になる。
CREATE INDEX index_exports_on_created_at_where_queued
    ON exports (created_at)
    WHERE status = 'queued';

CREATE INDEX index_exports_on_started_at_where_started
    ON exports (started_at)
    WHERE status = 'started';

-- migrate:down

DROP INDEX IF EXISTS index_exports_on_started_at_where_started;
DROP INDEX IF EXISTS index_exports_on_created_at_where_queued;
