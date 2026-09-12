-- name: MarkExportProfileDeletionStarted :one
-- Persist the deletion marker before waiting for in-flight export operations. The
-- existing value is kept on retries, while the UPDATE still takes the profile
-- row lock that serializes it with CreateExport's FOR SHARE gate.
--
-- [Ja] 進行中の export 操作を待つ前に削除マーカーを永続化する。再実行では既存値を
-- 維持しつつ、CreateExport の FOR SHARE ガードと直列化するプロフィール行ロックは
-- UPDATE によって取得する。
UPDATE profiles
SET export_deletion_started_at = COALESCE(export_deletion_started_at, NOW())
WHERE id = $1
RETURNING export_deletion_started_at;

-- name: GetExportProfileDeletionStartedAt :one
-- Read the persistent marker. Export operations call this before waiting for
-- the shared advisory lock and again while holding it, so post-deletion work
-- stops promptly without reopening the check-to-lock race. A missing profile
-- has no export work either and is represented by sql.ErrNoRows.
--
-- [Ja] 永続マーカーを読む。export 操作は共有 advisory lock を待つ前と取得後に
-- 呼び出すため、削除開始後の処理を速やかに止めつつ、確認と lock の間の競合も
-- 開け直さない。存在しないプロフィールにも export 作業は無く、sql.ErrNoRows として返す。
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
