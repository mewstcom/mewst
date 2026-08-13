-- migrate:up

-- Recreate the time-ordered recovery indexes with id as the tie-break key.
-- The recovery queries compare and order by the complete (timestamp, id)
-- keyset, so including both columns lets the cursor become the exact starting
-- point of each index scan even when many exports share the same timestamp.
--
-- [Ja] 時刻順の回復インデックスを、tie-break キーの id を含めて作り直す。
-- 回復クエリは完全な (timestamp, id) keyset で比較・整列するため、両方の
-- カラムを含めることで、同じ時刻を持つエクスポートが多数あっても cursor が
-- 各インデックス走査の正確な開始位置になる。
DROP INDEX index_exports_on_created_at_where_queued;
DROP INDEX index_exports_on_started_at_where_started;
DROP INDEX index_exports_on_finished_at_where_unnotified;

CREATE INDEX index_exports_on_created_at_where_queued
    ON exports (created_at, id)
    WHERE status = 'queued';

CREATE INDEX index_exports_on_started_at_where_started
    ON exports (started_at, id)
    WHERE status = 'started';

CREATE INDEX index_exports_on_finished_at_where_unnotified
    ON exports (finished_at, id)
    WHERE status = 'succeeded' AND completion_notified_at IS NULL;

-- migrate:down

DROP INDEX IF EXISTS index_exports_on_finished_at_where_unnotified;
DROP INDEX IF EXISTS index_exports_on_started_at_where_started;
DROP INDEX IF EXISTS index_exports_on_created_at_where_queued;

CREATE INDEX index_exports_on_created_at_where_queued
    ON exports (created_at)
    WHERE status = 'queued';

CREATE INDEX index_exports_on_started_at_where_started
    ON exports (started_at)
    WHERE status = 'started';

CREATE INDEX index_exports_on_finished_at_where_unnotified
    ON exports (finished_at)
    WHERE status = 'succeeded' AND completion_notified_at IS NULL;
