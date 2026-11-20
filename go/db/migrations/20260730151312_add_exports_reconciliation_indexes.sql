-- migrate:up

-- 完了メールが記録されないままのsucceededを探すリコンシリエーションの
-- クエリ用インデックス。部分インデックスの述語により定常状態ではほぼ空になる
-- (通知済みの行はcompletion_notified_atの打刻と同時にインデックスから外れる)
-- ため、periodic jobはsucceeded全件を走査せず小さなインデックスを引くだけで
-- 済む。
CREATE INDEX index_exports_on_finished_at_where_unnotified
    ON exports (finished_at)
    WHERE status = 'succeeded' AND completion_notified_at IS NULL;

-- succeededをprofileごとに集計し、2件以上持つプロフィールを探す
-- リコンシリエーションのクエリ用インデックス。profile_id順に並ぶため、集計は
-- テーブルを走査してハッシュする代わりにインデックスを順に読める。
CREATE INDEX index_exports_on_profile_id_where_succeeded
    ON exports (profile_id)
    WHERE status = 'succeeded';

-- migrate:down

DROP INDEX IF EXISTS index_exports_on_profile_id_where_succeeded;
DROP INDEX IF EXISTS index_exports_on_finished_at_where_unnotified;
