-- migrate:up

-- Snapshot the profile the notification belongs to, alongside the recipient
-- values already snapshotted here. Delivery has to decide whether the profile
-- is being deleted before it calls the mail provider, and reconciliation has to
-- leave the notifications of such a profile alone; both then read one row
-- instead of resolving the profile through the requester.
--
-- [Ja] 通知が属するプロフィールを、ここに既に snapshot されている宛先の値と並べて
-- snapshot する。配信はメールプロバイダーを呼ぶ前にプロフィールが削除中かを判断する
-- 必要があり、リコンシリエーションはそのプロフィールの通知を対象から外す必要がある。
-- どちらも申請者からプロフィールを解決せず、1 行を読むだけで済むようになる。
ALTER TABLE export_completion_notifications
    ADD COLUMN profile_id uuid;

UPDATE export_completion_notifications
SET profile_id = actors.profile_id
FROM actors
WHERE actors.id = export_completion_notifications.actor_id;

ALTER TABLE export_completion_notifications
    ALTER COLUMN profile_id SET NOT NULL;

-- Reference the requester and the profile as one pair, the way exports does, so
-- a row whose profile differs from its requester's cannot be stored. The
-- cascade stays on this constraint: deleting the requester still cancels an
-- email that is no longer useful, and the notification never blocks the
-- deletion of what it points at.
--
-- Both statements below lock actors, which Rails owns and writes to on every
-- sign-up: dropping the constraint removes referential triggers from it, and
-- adding one takes a share row exclusive lock on it while the new constraint is
-- validated. The validation reads the referencing table, and this one was
-- created in the previous task and holds at most one row per pending completion
-- email, so a plain ALTER TABLE is used. A larger referencing table would need
-- ADD CONSTRAINT ... NOT VALID and a separate VALIDATE CONSTRAINT instead.
--
-- [Ja] exports と同じく申請者とプロフィールを 1 組として参照し、申請者と別の
-- プロフィールを持つ行を保存できないようにする。CASCADE はこの制約に引き継ぐ。
-- 申請者を削除すれば不要になったメールは取り消され、通知が参照先の削除を妨げる
-- こともない。
--
-- 以下の 2 文はどちらも actors をロックする。actors は Rails が所有し、サインアップ
-- ごとに書き込むテーブルである。制約の削除は actors から参照整合性トリガーを外し、
-- 制約の追加は新しい制約の検証中に actors へ share row exclusive lock を取る。検証が
-- 読むのは参照側のテーブルであり、それは前タスクで作成されたばかりで、送信待ちの
-- 完了メール 1 件につき 1 行しか持たないため、通常の ALTER TABLE を使う。参照側が
-- 大きくなった場合は、代わりに ADD CONSTRAINT ... NOT VALID と別の
-- VALIDATE CONSTRAINT に分ける必要がある。
ALTER TABLE export_completion_notifications
    DROP CONSTRAINT export_completion_notifications_actor_id_fkey;

ALTER TABLE export_completion_notifications
    ADD CONSTRAINT export_completion_notifications_actor_profile_fkey
        FOREIGN KEY (actor_id, profile_id)
        REFERENCES actors (id, profile_id) ON DELETE CASCADE;

-- Profile deletion cancels the profile's pending notifications in one
-- statement. Without this index that delete reads the whole table.
--
-- Reaching the notifications through actors is what index_actors_on_profile_id
-- was created for, and this index replaces that path, but the actors one stays:
-- profile deletion still looks actors up by profile_id alone, both from the
-- Rails association check and from the foreign key check on actors.profile_id,
-- and the remaining composite index starts with user_id.
--
-- [Ja] プロフィールの削除は、そのプロフィールの送信待ち通知を 1 文で取り消す。この
-- インデックスが無いと、その削除はテーブル全体を読む。
--
-- 通知を actors 経由で辿る経路は index_actors_on_profile_id が作られた理由であり、
-- 本インデックスがその経路を置き換えるが、actors 側の索引は維持する。プロフィールの
-- 削除は Rails の関連確認と actors.profile_id の外部キー検査の双方で、依然として
-- profile_id 単体で actors を検索するためである。残る複合索引は user_id から始まる
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
