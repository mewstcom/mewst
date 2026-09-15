-- name: GetExportCompletionNotificationByExportID :one
SELECT * FROM export_completion_notifications
WHERE export_id = sqlc.arg(export_id);

-- name: ListPendingExportCompletionNotifications :many
-- thresholdより前に作成された送信待ち完了通知を古い順に返す。行はexportの
-- succeeded遷移と原子的に作成され、そのexportのcleanup後も残るため、senderが
-- 送信成功後に削除するまでdurable work intentになる。
--
-- 削除が始まったプロフィールの行は返さない。理由はListStaleQueuedExportsと同じで
-- ある。配信はそのプロフィールの削除マーカーで止まるため、再投入したジョブは行に
-- 触れずに戻り、マーカーが戻ることもない。除外しなければ、その行は毎回の実行で候補に
-- なり、新しい処理の予算を消費しながら何もしないジョブを投入し続ける。これらの行を
-- 収束させるのは親削除であって、リコンシリエーションではない。
--
-- 1ページ目はゼロ時刻とゼロUUIDを渡す。どちらも保存されるすべての行より前に
-- 並ぶため、同じ無条件のkeyset比較で1ページ目と後続ページを扱える。
SELECT export_completion_notifications.* FROM export_completion_notifications
WHERE export_completion_notifications.created_at < sqlc.arg(threshold)
  AND (export_completion_notifications.created_at, export_completion_notifications.export_id)
      > (sqlc.arg(after_time)::timestamptz, sqlc.arg(after_id)::uuid)
  AND NOT EXISTS (
    SELECT 1 FROM profiles
    WHERE profiles.id = export_completion_notifications.profile_id
      AND profiles.export_deletion_started_at IS NOT NULL
  )
ORDER BY export_completion_notifications.created_at ASC, export_completion_notifications.export_id ASC
LIMIT sqlc.arg(page_size);

-- name: MarkExportCompletionNotificationSent :one
-- 配信済みの完了メールのwork intentを退役させる。送信を記録するのはoutbox行の
-- 削除であり、legacyなcompletion_notified_at列は同じ文でmirrorする。outbox導入前の
-- スキーマへ切り戻しても、通知が完了済みに見えるようにするためである。
--
-- export行が無い場合、mirrorは行われない。保持cleanupは通知を意図的に残したまま
-- exportを削除するため、これが通常のケースになる。したがって行数はupdateではなく
-- deleteから取る。update側を読むと、outboxがまさに可能にしている送信に対して
-- 「退役させるものが無い」と答えてしまう。データ変更を伴うCTEは、主クエリが読むか
-- どうかに関わらず最後まで実行されるため、selectがdeletedだけを読んでいてもupdateは
-- 実行される。
--
-- statusの述語はexportsが打刻に対して要求しているものである。
-- exports_completion_notified_at_checkはsucceededの行にしかcompletion_notified_atを
-- 許さないため、これが無いと他の状態の行で文全体が失敗し、配信済みのメールが退役できなく
-- なる。
--
-- completion_notified_atはもはや通知状態の正本ではないため、打刻にupdated_atの更新を
-- 伴わせない。updated_atはexport自身の遷移における楽観ロックのトークンであり、ここで
-- 動かすとlegacyなmirrorが状態遷移のように見えてしまう。
WITH deleted AS (
    DELETE FROM export_completion_notifications
    WHERE export_id = sqlc.arg(export_id)
    RETURNING export_id
),
mirrored AS (
    UPDATE exports
    SET completion_notified_at = NOW()
    WHERE exports.id IN (SELECT export_id FROM deleted)
      AND exports.status = 'succeeded'
    RETURNING exports.id
)
SELECT count(*) FROM deleted;

-- name: DeleteExportCompletionNotificationsByProfileID :execrows
-- プロフィールの送信待ち完了メールをすべて取り消す。プロフィール自体の削除は
-- それらをこう扱うことになる。メールが知らせるアーカイブは消えており、宛先も削除
-- されようとしているからである。
--
-- 行はexportではなく通知にsnapshotされたプロフィールから辿る。export行は先に
-- 削除され、通知は意図的にそれより長く残るためである。
DELETE FROM export_completion_notifications
WHERE profile_id = sqlc.arg(profile_id);
