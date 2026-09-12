-- name: GetExportCompletionNotificationByExportID :one
SELECT * FROM export_completion_notifications
WHERE export_id = sqlc.arg(export_id);

-- name: ListPendingExportCompletionNotifications :many
-- Return pending completion notifications created before the threshold,
-- oldest first. The row is created atomically with the export succeeded
-- transition and survives cleanup of that export, so it remains the durable
-- work intent until the sender deletes it after a successful send.
--
-- Rows of a profile whose deletion has started are left out, for the same
-- reason as in ListStaleQueuedExports. Delivery stops at that profile's
-- deletion marker, so a re-enqueued job would return without touching the row,
-- and the marker is never cleared. Without the exclusion the row stays a
-- candidate on every run and keeps producing jobs that do nothing while
-- consuming the run's budget for new work. Converging these rows is profile
-- deletion's work, not reconciliation's.
--
-- The first page passes the zero timestamp and zero UUID. Both sort before
-- every stored row, letting one unconditional keyset comparison drive the
-- first and later pages.
--
-- [Ja] threshold より前に作成された送信待ち完了通知を古い順に返す。行は export の
-- succeeded 遷移と原子的に作成され、その export の cleanup 後も残るため、sender が
-- 送信成功後に削除するまで durable work intent になる。
--
-- 削除が始まったプロフィールの行は返さない。理由は ListStaleQueuedExports と同じで
-- ある。配信はそのプロフィールの削除マーカーで止まるため、再投入したジョブは行に
-- 触れずに戻り、マーカーが戻ることもない。除外しなければ、その行は毎回の実行で候補に
-- なり、新しい処理の予算を消費しながら何もしないジョブを投入し続ける。これらの行を
-- 収束させるのは親削除であって、リコンシリエーションではない。
--
-- 1 ページ目はゼロ時刻とゼロ UUID を渡す。どちらも保存されるすべての行より前に
-- 並ぶため、同じ無条件の keyset 比較で 1 ページ目と後続ページを扱える。
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
-- Retire the work intent of a completion email that was delivered. Deleting the
-- outbox row is what records the send, and the legacy completion_notified_at
-- column is mirrored in the same statement so that a rollback to the schema
-- before the outbox still sees the notification as done.
--
-- The mirror is skipped when the export row is gone, which retention cleanup
-- makes the normal case: it deletes the export while the notification
-- deliberately outlives it. The row count therefore comes from the delete, not
-- from the update, because reading the update would answer "nothing to retire"
-- for exactly the sends the outbox exists to keep possible. Data-modifying CTEs
-- run to completion whether or not the primary query reads them, so the update
-- still runs even though the select reads deleted alone.
--
-- The status predicate is what exports requires of the stamp:
-- exports_completion_notified_at_check only allows completion_notified_at on a
-- succeeded row, so without it a row in any other state would fail the whole
-- statement and leave a delivered email unretired.
--
-- completion_notified_at is not the notification state any more, so its stamp is
-- not paired with a bump of updated_at: that column is the optimistic-lock token
-- of the export's own transitions, and moving it here would make a legacy mirror
-- look like a state change.
--
-- [Ja] 配信済みの完了メールの work intent を退役させる。送信を記録するのは outbox 行の
-- 削除であり、legacy な completion_notified_at 列は同じ文で mirror する。outbox 導入前の
-- スキーマへ切り戻しても、通知が完了済みに見えるようにするためである。
--
-- export 行が無い場合、mirror は行われない。保持 cleanup は通知を意図的に残したまま
-- export を削除するため、これが通常のケースになる。したがって行数は update ではなく
-- delete から取る。update 側を読むと、outbox がまさに可能にしている送信に対して
-- 「退役させるものが無い」と答えてしまう。データ変更を伴う CTE は、主クエリが読むか
-- どうかに関わらず最後まで実行されるため、select が deleted だけを読んでいても update は
-- 実行される。
--
-- status の述語は exports が打刻に対して要求しているものである。
-- exports_completion_notified_at_check は succeeded の行にしか completion_notified_at を
-- 許さないため、これが無いと他の状態の行で文全体が失敗し、配信済みのメールが退役できなく
-- なる。
--
-- completion_notified_at はもはや通知状態の正本ではないため、打刻に updated_at の更新を
-- 伴わせない。updated_at は export 自身の遷移における楽観ロックのトークンであり、ここで
-- 動かすと legacy な mirror が状態遷移のように見えてしまう。
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
-- Cancel every pending completion email of the profile, which is what deleting
-- the profile itself makes of them: the archive the email announces is gone and
-- the address it would reach is being removed.
--
-- The rows are found through the profile snapshotted on the notification rather
-- than through the export, because the export row is deleted first and the
-- notification deliberately outlives it.
--
-- [Ja] プロフィールの送信待ち完了メールをすべて取り消す。プロフィール自体の削除は
-- それらをこう扱うことになる。メールが知らせるアーカイブは消えており、宛先も削除
-- されようとしているからである。
--
-- 行は export ではなく通知に snapshot されたプロフィールから辿る。export 行は先に
-- 削除され、通知は意図的にそれより長く残るためである。
DELETE FROM export_completion_notifications
WHERE profile_id = sqlc.arg(profile_id);
