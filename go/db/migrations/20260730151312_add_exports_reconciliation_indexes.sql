-- migrate:up

-- Index for the reconciliation query that finds succeeded exports whose
-- completion email was never recorded. The partial predicate keeps the index
-- nearly empty in steady state (a row leaves the index as soon as
-- completion_notified_at is stamped), so the periodic job probes a small index
-- instead of scanning every succeeded row.
--
-- [Ja] 完了メールが記録されないままの succeeded を探すリコンシリエーションの
-- クエリ用インデックス。部分インデックスの述語により定常状態ではほぼ空になる
-- (通知済みの行は completion_notified_at の打刻と同時にインデックスから外れる)
-- ため、periodic job は succeeded 全件を走査せず小さなインデックスを引くだけで
-- 済む。
CREATE INDEX index_exports_on_finished_at_where_unnotified
    ON exports (finished_at)
    WHERE status = 'succeeded' AND completion_notified_at IS NULL;

-- Index for the reconciliation query that groups succeeded exports by profile
-- to find the profiles holding more than one. Ordering by profile_id lets the
-- grouping read the index in order instead of scanning and hashing the table.
--
-- [Ja] succeeded を profile ごとに集計し、2 件以上持つプロフィールを探す
-- リコンシリエーションのクエリ用インデックス。profile_id 順に並ぶため、集計は
-- テーブルを走査してハッシュする代わりにインデックスを順に読める。
CREATE INDEX index_exports_on_profile_id_where_succeeded
    ON exports (profile_id)
    WHERE status = 'succeeded';

-- migrate:down

DROP INDEX IF EXISTS index_exports_on_profile_id_where_succeeded;
DROP INDEX IF EXISTS index_exports_on_finished_at_where_unnotified;
