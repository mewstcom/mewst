-- migrate:up

-- 完了メールの処理をexport行から独立して保持する。旧succeeded exportは
-- 新しいアーカイブが利用可能になるとすぐ削除されるが、その完了メールはcleanup
-- 後も再試行できなければならない。export成功時に宛先をsnapshotし、senderが
-- 削除済みexport行へ依存せず解決できるようにする。actor外部キーのCASCADEは
-- 意図的で、申請者の削除時には不要になったメールも取り消す。
CREATE TABLE export_completion_notifications (
    export_id uuid NOT NULL PRIMARY KEY,
    actor_id uuid NOT NULL
        REFERENCES actors (id) ON DELETE CASCADE,
    recipient_email citext NOT NULL,
    locale VARCHAR NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- pending行自体をdurable work intentとする。リコンシリエーションは作成順に
-- 走査し、senderは送信成功後にだけ削除する。同時刻の行も完全なkeyset順序で
-- 走査できるようexport_idを含める。
CREATE INDEX index_export_completion_notifications_on_created_at_and_id
    ON export_completion_notifications (created_at, export_id);

-- actor外部キーはCASCADEのため、申請者の削除では参照している行を探す必要が
-- ある。このインデックスが無いと、その検索はactorの削除のたびにテーブル全体を
-- 読む。
CREATE INDEX index_export_completion_notifications_on_actor_id
    ON export_completion_notifications (actor_id);

-- プロフィール削除では、そのプロフィールの全actorを検索して送信待ち通知を
-- 取り消す。actorsの主キーと一意インデックスはidまたはuser_idから始まり、
-- この検索には使えないため、profile_id専用のアクセスパスを設ける。
CREATE INDEX index_actors_on_profile_id
    ON actors (profile_id);

-- このテーブルより前に生まれたpending workを保持する。finished_atは通知が
-- 対象になった時刻で、既存の回復順序も維持できる。completion_notified_atは
-- downgrade互換用に残す。outboxを正本とし、legacy列が削除されるまで、互換目的で
-- 送信成功だけをこの列へmirrorする可能性がある。
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
