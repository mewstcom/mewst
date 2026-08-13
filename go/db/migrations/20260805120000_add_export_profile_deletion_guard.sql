-- migrate:up

-- Persist the start of profile deletion so no export can be created or
-- generated in the gap between archive cleanup and removal of the profile
-- row. The marker remains until the profile is deleted, making retries keep
-- the same closed boundary after a partial failure.
--
-- [Ja] アーカイブの cleanup とプロフィール行の削除の間に新しい export が作成・
-- 生成されないよう、プロフィール削除の開始を永続化する。部分失敗後の再実行でも
-- 同じ閉じた境界を保つため、マーカーはプロフィールが削除されるまで残す。
ALTER TABLE profiles
    ADD COLUMN export_deletion_started_at TIMESTAMP WITH TIME ZONE;

-- migrate:down

ALTER TABLE profiles
    DROP COLUMN IF EXISTS export_deletion_started_at;
