-- name: MarkExportProfileDeletionStarted :one
-- 進行中のexport操作を待つ前に削除マーカーを永続化する。再実行では既存値を
-- 維持しつつ、CreateExportのFOR SHAREガードと直列化するプロフィール行ロックは
-- UPDATEによって取得する。
UPDATE profiles
SET export_deletion_started_at = COALESCE(export_deletion_started_at, NOW())
WHERE id = $1
RETURNING export_deletion_started_at;

-- name: GetExportProfileDeletionStartedAt :one
-- 永続マーカーを読む。export操作は共有advisory lockを待つ前と取得後に
-- 呼び出すため、削除開始後の処理を速やかに止めつつ、確認とlockの間の競合も
-- 開け直さない。存在しないプロフィールにもexport作業は無く、sql.ErrNoRowsとして返す。
SELECT export_deletion_started_at
FROM profiles
WHERE id = $1;

-- name: AcquireExportProfileOperationLock :exec
SELECT pg_advisory_lock_shared($1::bigint);

-- name: ReleaseExportProfileOperationLock :one
SELECT pg_advisory_unlock_shared($1::bigint);

-- name: AcquireExportProfileDeletionLock :exec
SELECT pg_advisory_lock($1::bigint);

-- name: ReleaseExportProfileDeletionLock :one
SELECT pg_advisory_unlock($1::bigint);
