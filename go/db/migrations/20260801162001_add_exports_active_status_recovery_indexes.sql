-- migrate:up

-- 滞留したqueued / startedのエクスポートを走査するリコンシリエーションの
-- クエリ用インデックス。activeな行に対する部分ユニークインデックスは走査を進行中の
-- エクスポートに限定するが、キーがprofile_idのため並び順のカラムを持たず、どの
-- ページでもactiveな行を全件読んでソートすることになる。並び順のカラムをキーに
-- することでkeyset cursorが索引スキャンの開始位置になり、バックログが積み上がった
-- ときに1ページあたりの仕事量が有界になる。
CREATE INDEX index_exports_on_created_at_where_queued
    ON exports (created_at)
    WHERE status = 'queued';

CREATE INDEX index_exports_on_started_at_where_started
    ON exports (started_at)
    WHERE status = 'started';

-- migrate:down

DROP INDEX IF EXISTS index_exports_on_started_at_where_started;
DROP INDEX IF EXISTS index_exports_on_created_at_where_queued;
