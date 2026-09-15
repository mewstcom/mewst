-- migrate:up

-- アーカイブのcleanupとプロフィール行の削除の間に新しいexportが作成・
-- 生成されないよう、プロフィール削除の開始を永続化する。部分失敗後の再実行でも
-- 同じ閉じた境界を保つため、マーカーはプロフィールが削除されるまで残す。
ALTER TABLE profiles
    ADD COLUMN export_deletion_started_at TIMESTAMP WITH TIME ZONE;

-- migrate:down

ALTER TABLE profiles
    DROP COLUMN IF EXISTS export_deletion_started_at;
