-- name: CreateExport :one
-- exportの作成とkept投稿の固定化を1つのPostgreSQL文で行う。2つの
-- data-modifying CTEは同じstatement snapshotを使うため、申請より後にcommit
-- された投稿が入り込んだり、後から物理削除された投稿が抜け落ちたりしない。
-- data-modifying CTEは主問い合わせがその出力を読むかどうかに関係なく、ちょうど
-- 1回、完了まで実行されるため、複製のために最後のSELECTから
-- snapshotted_postsを参照する必要はない。
WITH profile_gate AS MATERIALIZED (
    -- 永続的な削除マーカーを読む前にプロフィール行をロックする。FOR SHAREは
    -- 削除開始のUPDATEと競合するため、先にロックしたCreateは完了後にcleanupの
    -- 対象となり、削除を待ったCreateは新しいマーカーを見て何もINSERTしない。
    SELECT profiles.id, profiles.export_deletion_started_at
    FROM profiles
    WHERE profiles.id = sqlc.arg(profile_id)
    FOR SHARE
),
created_export AS (
    INSERT INTO exports (profile_id, actor_id)
    SELECT sqlc.arg(profile_id), sqlc.arg(actor_id)
    WHERE NOT EXISTS (
        SELECT 1
        FROM profile_gate
        WHERE profile_gate.export_deletion_started_at IS NOT NULL
    )
    RETURNING *
),
snapshotted_posts AS (
    INSERT INTO export_posts (export_id, post_id, content, published_at)
    SELECT
        created_export.id,
        posts.id,
        posts.content,
        posts.published_at
    FROM created_export
    JOIN posts ON posts.profile_id = created_export.profile_id
    WHERE posts.discarded_at IS NULL
    RETURNING export_id
)
SELECT * FROM created_export;

-- name: GetExportByID :one
SELECT * FROM exports
WHERE id = $1;

-- name: GetLatestExportByProfileID :one
SELECT * FROM exports
WHERE profile_id = $1
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: GetLatestSucceededExportByProfileID :one
SELECT * FROM exports
WHERE profile_id = $1 AND status = 'succeeded'
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: MarkExportStarted :one
-- 4つの状態遷移 (MarkExportStarted / MarkExportSucceeded /
-- MarkExportFailed / RequeueExport) は呼び出し側の期待するupdated_atを
-- ガードにするため、updated_atは楽観ロックのトークンを兼ねており、更新のたびに
-- 厳密増加させる必要がある。NOW() はトランザクション内で固定され、
-- clock_timestamp() 単独でもマイクロ秒解像度で同値になり得るため、
-- GREATEST(clock_timestamp(), updated_at + INTERVAL '1 microsecond') で必ず
-- より大きいトークンを保証する。NOW() や素のclock_timestamp() へ単純化しない
-- こと。古い試行がガードを通過し得る。
--
-- 行数ではなく更新後の行を返すのは、生成処理がこの遷移を次のガード付き更新へ
-- つなぐため。同じ試行をMarkExportSucceeded / MarkExportFailedで終わらせるには、
-- 本文が生成したトークンを知る必要がある。別の文で行を読み直すとトークンが閉じて
-- いる窓をふたたび開くことになる。その間にcommitされた遷移があれば、呼び出し側は
-- 自分が保持していないトークンを受け取ってしまうため。
UPDATE exports
SET status = 'started',
    started_at = NOW(),
    attempt_count = attempt_count + 1,
    updated_at = GREATEST(clock_timestamp(), updated_at + INTERVAL '1 microsecond')
WHERE id = sqlc.arg(id)
  AND status IN ('queued', 'started')
  AND updated_at = sqlc.arg(expected_updated_at)
RETURNING *;

-- name: MarkExportSucceeded :execrows
-- updated_at式は楽観ロックのトークン。厳密増加させる理由は
-- MarkExportStartedを参照。
--
-- succeededへの到達では、完了通知のwork intentを作成し、申請時の投稿snapshot
-- も同じ文で破棄する。通知にはプロフィール・申請者のメールアドレス・localeを
-- snapshotするため、cleanupがexport行を削除してもpending workは失われず、配信は
-- 通知だけでプロフィールの削除境界に対する判断ができる。投稿snapshotは再試行が
-- 初回と同じ入力を読むために存在し、アーカイブのupload後は不要になる。3つの変更を
-- 原子的に行うことで、通知の無いsucceededや、行と投稿snapshotの食い違いを防ぐ。
--
-- 最後のselectがmarkedを読むのは、行数が呼び出し側のガード対象である遷移だけを
-- 表すようにするため。遷移が成立すれば通知は必ず作成される。actor_idと
-- actors.user_idはどちらもNOT NULLの外部キーであり、結合が空になり得ないからで
-- ある。insert側を読むと、結合が失敗したときに「遷移しなかった」と答えてしまう。
WITH marked AS (
    UPDATE exports
    SET status = 'succeeded',
        object_key = sqlc.arg(object_key),
        finished_at = NOW(),
        updated_at = GREATEST(clock_timestamp(), updated_at + INTERVAL '1 microsecond')
    WHERE exports.id = sqlc.arg(id)
      AND exports.status = 'started'
      AND exports.updated_at = sqlc.arg(expected_updated_at)
    RETURNING id, actor_id, finished_at
),
created_notification AS (
    INSERT INTO export_completion_notifications (
        export_id,
        actor_id,
        profile_id,
        recipient_email,
        locale,
        created_at
    )
    SELECT
        marked.id,
        marked.actor_id,
        actors.profile_id,
        users.email,
        users.locale,
        marked.finished_at
    FROM marked
    JOIN actors ON actors.id = marked.actor_id
    JOIN users ON users.id = actors.user_id
    RETURNING export_id
),
discarded_snapshot AS (
    DELETE FROM export_posts
    WHERE export_id IN (SELECT id FROM marked)
)
SELECT id FROM marked;

-- name: MarkExportFailed :execrows
-- updated_at式は楽観ロックのトークン。厳密増加させる理由は
-- MarkExportStartedを参照。
--
-- failedは終端状態のため、MarkExportSucceededと同じ理由で申請時の投稿snapshotを
-- 同じ文で破棄する。以後それを読むものは無い。
WITH marked AS (
    UPDATE exports
    SET status = 'failed',
        finished_at = NOW(),
        updated_at = GREATEST(clock_timestamp(), updated_at + INTERVAL '1 microsecond')
    WHERE id = sqlc.arg(id)
      AND status = 'started'
      AND updated_at = sqlc.arg(expected_updated_at)
    RETURNING id
),
discarded_snapshot AS (
    DELETE FROM export_posts
    WHERE export_id IN (SELECT id FROM marked)
)
SELECT id FROM marked;

-- name: RequeueExport :execrows
-- updated_at式は楽観ロックのトークン。厳密増加させる理由は
-- MarkExportStartedを参照。
UPDATE exports
SET status = 'queued',
    started_at = NULL,
    updated_at = GREATEST(clock_timestamp(), updated_at + INTERVAL '1 microsecond')
WHERE id = sqlc.arg(id)
  AND status = 'started'
  AND updated_at = sqlc.arg(expected_updated_at);

-- name: ListStaleQueuedExports :many
-- thresholdより前に作成されたqueuedのエクスポートを古い順に返す。生成
-- ジョブはCreateのコミット直後に投入されるため、作成から時間が経ってもまだ
-- queuedの行はRiverへの投入が起きなかったこと (コミットと投入の間でプロセスが
-- 落ちた、または投入が失敗した) を意味する。thresholdは通常の投入がまだ処理中の
-- 行を再投入しないための猶予期間。リスクの窓はCreate時点で開くためupdated_at
-- ではなくcreated_atを使う。再投入は一意ジョブにより冪等。
--
-- 削除が始まったプロフィールの行は返さない。生成はそのプロフィールの削除マーカーで
-- 止まるため、再投入したジョブは行に触れずに戻り、マーカーが戻ることもない。除外し
-- なければ、その行は毎回の実行で候補になり、何もしないジョブを投入し続ける。これら
-- の行を収束させるのは親削除であって、リコンシリエーションではない。
--
-- cursorとpage sizeは1クエリの取得量を抑えつつ、毎回同じバックログの
-- 先頭へ固定されるのを防ぐ。リコンシリエーションは既存ジョブの候補をcursorで
-- 飛ばし、新しい処理を1回の予算まで受理した時点で止まるため、先頭の停滞が後続
-- 行を飢えさせない。
--
-- 1ページ目はゼロ時刻とゼロUUIDを渡す。どちらも保存されるどの行よりも前に
-- 並ぶため、1つの無条件な比較で1ページ目と2ページ目以降の両方をまかなえる。
-- 「cursorの有無」フラグとORで包むと、パラメータ値がプラン時に未知の場合に
-- cursorを索引スキャンの開始位置として使えなくなる。
SELECT exports.* FROM exports
WHERE exports.status = 'queued'
  AND exports.created_at < sqlc.arg(threshold)
  AND (exports.created_at, exports.id) > (sqlc.arg(after_time)::timestamptz, sqlc.arg(after_id)::uuid)
  AND NOT EXISTS (
    SELECT 1 FROM profiles
    WHERE profiles.id = exports.profile_id
      AND profiles.export_deletion_started_at IS NOT NULL
  )
ORDER BY exports.created_at ASC, exports.id ASC
LIMIT sqlc.arg(page_size);

-- name: ListStaleStartedExports :many
-- 現在の試行がthresholdより前に始まったstartedのエクスポートを古い順に
-- 返す。started_atはMarkExportStartedのたび (リトライを含む) に打刻されるため
-- 実行中の試行の開始時刻を表す。タイムアウトと猶予期間を足した時間より古い
-- startedの行は、Workerが後処理に到達せず落ちたことを意味する。呼び出し側が
-- 再投入 (attempt_countが上限未満) とfailed (上限到達) を判断する。
--
-- 削除が始まったプロフィールの行は、ListStaleQueuedExportsと同じ理由で返さない。
-- 生成はそのプロフィールの削除マーカーで止まるため、差し戻した行は、それに触れずに
-- 戻るジョブを生むだけである。ここでの無駄は無期限ではなく有界である。差し戻しは行を
-- queuedへ移し、queuedの系統は既にその行を除外しているためである。除外することで、
-- その予算をまだ収束し得る処理に充て、3つの回復系統でルールを揃える。
--
-- cursorにより、リコンシリエーションは停滞中または投入済みの先頭候補を
-- 有界なページで飛ばし、新しい処理の予算を別に制御できる。1ページ目が
-- 「cursorの有無」フラグではなくゼロ値を渡す理由はListStaleQueuedExportsを
-- 参照。
SELECT * FROM exports
WHERE status = 'started'
  AND started_at < sqlc.arg(threshold)
  AND (started_at, id) > (sqlc.arg(after_time)::timestamptz, sqlc.arg(after_id)::uuid)
  AND NOT EXISTS (
    SELECT 1 FROM profiles
    WHERE profiles.id = exports.profile_id
      AND profiles.export_deletion_started_at IS NOT NULL
  )
ORDER BY started_at ASC, id ASC
LIMIT sqlc.arg(page_size);

-- name: ListOldSucceededExportsByProfileID :many
-- プロフィールのsucceededのうち最新の1件を除いたすべてを返し、cleanupが
-- R2オブジェクトと行を削除できるようにする。行値比較 (created_at, id) < (最新の
-- created_at, 最新のid) により、created_atが同値のときのtie-breakがidに
-- 落ち、最新を選ぶDESCの並びと一致する。厳密な < により最新のsucceeded自身は
-- 除外され、削除対象に選ばれない。
--
-- page sizeはリコンシリエーションのクエリと同じく1クエリの取得量を抑える。
-- cursorは不要で、cleanupは処理した行を削除するため、古い順の並びにより次回の
-- 実行で残りの候補が先頭に現れる。
SELECT e.* FROM exports e
WHERE e.profile_id = sqlc.arg(profile_id)
  AND e.status = 'succeeded'
  AND (e.created_at, e.id) < (
    SELECT latest.created_at, latest.id FROM exports latest
    WHERE latest.profile_id = sqlc.arg(profile_id) AND latest.status = 'succeeded'
    ORDER BY latest.created_at DESC, latest.id DESC
    LIMIT 1
  )
ORDER BY e.created_at ASC, e.id ASC
LIMIT sqlc.arg(page_size);

-- name: ListProfileIDsWithOldSucceededExports :many
-- succeededのエクスポートを2件以上持つ (= 掃除すべき古いsucceededの
-- 行がある) プロフィールIDを返す。リコンシリエーションが返された各プロフィール
-- ごとに一意なcleanupジョブを1件投入し、成功後のcleanup投入消失に対する
-- 安全網とする。
--
-- profile IDのcursorにより、リコンシリエーションは一意なcleanupジョブが
-- すでに存在するプロフィールを飛ばせる。page sizeは各クエリの取得量を抑え、
-- 呼び出し側は1回で受理する新しいcleanupジョブ数を別に制限する。1ページ目は
-- ゼロUUIDを渡す。保存されるどのprofile IDよりも前に並ぶためで、
-- 「cursorの有無」フラグより優れる理由はListStaleQueuedExportsを参照。
SELECT profile_id FROM exports
WHERE status = 'succeeded'
  AND profile_id > sqlc.arg(after_profile_id)
GROUP BY profile_id
HAVING COUNT(*) > 1
ORDER BY profile_id ASC
LIMIT sqlc.arg(page_size);

-- name: ListExportIDsRetainingObject :many
-- 与えたIDのうち、オブジェクトストレージ上のオブジェクトをまだ保持している
-- エクスポートだけを返す。孤児回収が、どのR2オブジェクト (キーはエクスポートID)
-- がまだ保持されているかを判別するために使う。結果に無いIDが孤児の候補。
--
-- failed以外のすべてのstatusが保持側になる。queued / startedのエクスポートは、
-- 前の試行がアップロードし現在の試行が上書きしようとしているオブジェクトを保持
-- しうる。succeededのエクスポートはダウンロード対象のアーカイブを保持する。
-- failedのエクスポートは何も保持しない。オブジェクトを手放すのが終端遷移であり、
-- failedの行の下に残ったオブジェクトはまさに孤児回収が回収するものであるため。
SELECT id FROM exports
WHERE id = ANY(sqlc.arg(ids)::uuid[])
  AND status <> 'failed';

-- name: ListExportsByProfileID :many
-- プロフィールのエクスポートをstatusを問わず古い順に1ページ返す。
-- 外部キーがON DELETE NO ACTIONであるため、プロフィールの削除はその
-- エクスポートをアプリケーション経由で削除する必要がある。行はエクスポートの
-- すべてではなく、オブジェクトストレージ上のオブジェクトはDBのCASCADEが
-- 及ぶ範囲の外にあるからである。
--
-- succeededだけでなく全statusを返す。それを記録する遷移より先にアップロードを
-- 終えた試行は、queued / started / failedの行の下にもオブジェクトを残すため、
-- 削除されるプロフィールがそれを残していってはならない。
--
-- page sizeはcleanupのクエリと同じく1クエリの取得量を抑え、cursorが不要な
-- 理由も同じである。呼び出し側は処理した行を削除するため、古い順の並びにより次の
-- クエリで残りが現れる。
SELECT * FROM exports
WHERE profile_id = sqlc.arg(profile_id)
ORDER BY created_at ASC, id ASC
LIMIT sqlc.arg(page_size);

-- name: DeleteExport :execrows
-- ID指定でエクスポート行を1件削除する。cleanupはR2オブジェクトが
-- 消えた後にこれを呼ぶため、オブジェクトが残ったまま行が消えることはない。
DELETE FROM exports
WHERE id = sqlc.arg(id);

-- name: DeleteFailedExportsByProfileID :execrows
-- プロフィールのfailedなエクスポートを削除する。Createは新しいqueuedの
-- エクスポートを挿入するのと同じtransactionでこれを呼ぶため、プロフィールが
-- 保持するのは最新の成功1件と、進行中またはfailedのエクスポート1件までになる。
--
-- 削除するのはfailedの行だけである。queued / startedの行はプロフィールの実行中の
-- エクスポートで部分ユニークインデックスが守っており、succeededの行は次の成功が
-- 置き換えるまでダウンロードできるアーカイブであるため。
--
-- failedのエクスポートはobject_keyを持たず (状態フィールドのCHECK制約が保証)、
-- 終端遷移がアップロード済みのオブジェクトを既に手放しているため、行を消しても
-- 取り残しは生じない。遷移より後まで残ったオブジェクトはfailedの行に保持されて
-- おらず、孤児回収が回収する。
DELETE FROM exports
WHERE profile_id = sqlc.arg(profile_id)
  AND status = 'failed';
