-- migrate:up

-- Keep completion-email work independently of the export row. Old succeeded
-- exports are removed as soon as a newer archive is available, but their
-- completion email must remain retryable after that cleanup. Recipient values
-- are snapshotted when the export succeeds so the sender never needs the
-- deleted export row to resolve them. The actor foreign key deliberately
-- cascades: deleting the requester cancels an email that is no longer useful.
--
-- [Ja] 完了メールの処理を export 行から独立して保持する。旧 succeeded export は
-- 新しいアーカイブが利用可能になるとすぐ削除されるが、その完了メールは cleanup
-- 後も再試行できなければならない。export 成功時に宛先を snapshot し、sender が
-- 削除済み export 行へ依存せず解決できるようにする。actor 外部キーの CASCADE は
-- 意図的で、申請者の削除時には不要になったメールも取り消す。
CREATE TABLE export_completion_notifications (
    export_id uuid NOT NULL PRIMARY KEY,
    actor_id uuid NOT NULL
        REFERENCES actors (id) ON DELETE CASCADE,
    recipient_email citext NOT NULL,
    locale VARCHAR NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- The pending row itself is the durable work intent. Reconciliation walks it
-- in creation order and the sender deletes it only after a successful send.
-- Including export_id completes the keyset order for equal timestamps.
--
-- [Ja] pending 行自体を durable work intent とする。リコンシリエーションは作成順に
-- 走査し、sender は送信成功後にだけ削除する。同時刻の行も完全な keyset 順序で
-- 走査できるよう export_id を含める。
CREATE INDEX index_export_completion_notifications_on_created_at_and_id
    ON export_completion_notifications (created_at, export_id);

-- The actor foreign key cascades, so deleting a requester has to find the rows
-- that reference it. Without this index that lookup reads the whole table on
-- every actor deletion.
--
-- [Ja] actor 外部キーは CASCADE のため、申請者の削除では参照している行を探す必要が
-- ある。このインデックスが無いと、その検索は actor の削除のたびにテーブル全体を
-- 読む。
CREATE INDEX index_export_completion_notifications_on_actor_id
    ON export_completion_notifications (actor_id);

-- Profile deletion cancels pending notifications by finding every actor of the
-- profile. The actors primary and uniqueness indexes start with id or user_id
-- and cannot serve that lookup, so profile_id needs its own access path.
--
-- [Ja] プロフィール削除では、そのプロフィールの全 actor を検索して送信待ち通知を
-- 取り消す。actors の主キーと一意インデックスは id または user_id から始まり、
-- この検索には使えないため、profile_id 専用のアクセスパスを設ける。
CREATE INDEX index_actors_on_profile_id
    ON actors (profile_id);

-- Preserve pending work that predates this table. finished_at is the instant
-- the notification became eligible and keeps the existing recovery order.
-- completion_notified_at is retained for downgrade compatibility; the outbox
-- remains authoritative, and successful delivery may only be mirrored here
-- for compatibility until the legacy column is removed.
--
-- [Ja] このテーブルより前に生まれた pending work を保持する。finished_at は通知が
-- 対象になった時刻で、既存の回復順序も維持できる。completion_notified_at は
-- downgrade 互換用に残す。outbox を正本とし、legacy 列が削除されるまで、互換目的で
-- 送信成功だけをこの列へ mirror する可能性がある。
INSERT INTO export_completion_notifications (
    export_id,
    actor_id,
    recipient_email,
    locale,
    created_at
)
SELECT
    exports.id,
    exports.actor_id,
    users.email,
    users.locale,
    exports.finished_at
FROM exports
JOIN actors ON actors.id = exports.actor_id
JOIN users ON users.id = actors.user_id
WHERE exports.status = 'succeeded'
  AND exports.completion_notified_at IS NULL;

-- migrate:down

DROP TABLE IF EXISTS export_completion_notifications;
DROP INDEX IF EXISTS index_actors_on_profile_id;
