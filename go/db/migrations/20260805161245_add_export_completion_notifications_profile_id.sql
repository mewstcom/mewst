-- migrate:up

-- 通知が属するプロフィールを、ここに既にsnapshotされている宛先の値と並べて
-- snapshotする。配信はメールプロバイダーを呼ぶ前にプロフィールが削除中かを判断する
-- 必要があり、リコンシリエーションはそのプロフィールの通知を対象から外す必要がある。
-- どちらも申請者からプロフィールを解決せず、1行を読むだけで済むようになる。
ALTER TABLE export_completion_notifications
    ADD COLUMN profile_id uuid;

UPDATE export_completion_notifications
SET profile_id = actors.profile_id
FROM actors
WHERE actors.id = export_completion_notifications.actor_id;

ALTER TABLE export_completion_notifications
    ALTER COLUMN profile_id SET NOT NULL;

-- exportsと同じく申請者とプロフィールを1組として参照し、申請者と別の
-- プロフィールを持つ行を保存できないようにする。CASCADEはこの制約に引き継ぐ。
-- 申請者を削除すれば不要になったメールは取り消され、通知が参照先の削除を妨げる
-- こともない。
--
-- 以下の2文はどちらもactorsをロックする。actorsはRailsが所有し、サインアップ
-- ごとに書き込むテーブルである。制約の削除はactorsから参照整合性トリガーを外し、
-- 制約の追加は新しい制約の検証中にactorsへshare row exclusive lockを取る。検証が
-- 読むのは参照側のテーブルであり、それは前タスクで作成されたばかりで、送信待ちの
-- 完了メール1件につき1行しか持たないため、通常のALTER TABLEを使う。参照側が
-- 大きくなった場合は、代わりにADD CONSTRAINT ... NOT VALIDと別の
-- VALIDATE CONSTRAINTに分ける必要がある。
ALTER TABLE export_completion_notifications
    DROP CONSTRAINT export_completion_notifications_actor_id_fkey;

ALTER TABLE export_completion_notifications
    ADD CONSTRAINT export_completion_notifications_actor_profile_fkey
        FOREIGN KEY (actor_id, profile_id)
        REFERENCES actors (id, profile_id) ON DELETE CASCADE;

-- プロフィールの削除は、そのプロフィールの送信待ち通知を1文で取り消す。この
-- インデックスが無いと、その削除はテーブル全体を読む。
--
-- 通知をactors経由で辿る経路はindex_actors_on_profile_idが作られた理由であり、
-- 本インデックスがその経路を置き換えるが、actors側の索引は維持する。プロフィールの
-- 削除はRailsの関連確認とactors.profile_idの外部キー検査の双方で、依然として
-- profile_id単体でactorsを検索するためである。残る複合索引はuser_idから始まる
-- ためこれを代替できない。
CREATE INDEX index_export_completion_notifications_on_profile_id
    ON export_completion_notifications (profile_id);

-- migrate:down

DROP INDEX IF EXISTS index_export_completion_notifications_on_profile_id;

ALTER TABLE export_completion_notifications
    DROP CONSTRAINT export_completion_notifications_actor_profile_fkey;

ALTER TABLE export_completion_notifications
    ADD CONSTRAINT export_completion_notifications_actor_id_fkey
        FOREIGN KEY (actor_id) REFERENCES actors (id) ON DELETE CASCADE;

ALTER TABLE export_completion_notifications
    DROP COLUMN profile_id;
