-- name: ListExportPostMonthsByExportID :many
-- exportの不変な投稿snapshotに含まれる暦月ごとに1行を、投稿件数と
-- その投稿を含むUTC走査範囲とともに古い順で返す。
-- エクスポートは月ごとに1つのHTMLファイルを書き、最初のファイルより前に
-- 件数を必要とするため、先にサマリーを取得し、その後で同じ固定済みsnapshot
-- から各月をその範囲で分割取得する。
--
-- export_posts.published_atはUTCを保持するtimestamp without time zone
-- のため、投稿が属する月はUTCとして読んでから対象タイムゾーンへ変換し、
-- truncateして求める。夏時間のフォールド中はローカル月初が曖昧になりうるため、
-- その壁時計を1つのUTC時刻へ逆変換しても安全な走査境界にはならない。
-- 代わりに、グループの正確なexport_posts行からMIN / MAXで狭い半開区間を
-- 導出する。分割取得クエリは正しさを守るためローカル月の述語を再適用し、
-- 範囲はインデックス走査に使う。
WITH months AS (
    SELECT
        date_trunc('month', (published_at AT TIME ZONE 'UTC') AT TIME ZONE sqlc.arg(time_zone)::text) AS local_month_start,
        MIN(published_at) AS starts_at,
        MAX(published_at) + INTERVAL '1 microsecond' AS ends_at,
        COUNT(*) AS post_count
    FROM export_posts
    WHERE export_id = sqlc.arg(export_id)
    GROUP BY 1
)
SELECT
    local_month_start::timestamp AS local_month_start,
    starts_at::timestamp AS starts_at,
    ends_at::timestamp AS ends_at,
    post_count::bigint AS post_count
FROM months
ORDER BY local_month_start ASC;

-- name: ListExportPostsByExportIDInRange :many
-- exportの不変な投稿snapshotのうち、半開区間のUTC走査範囲
-- [starts_at, ends_at) に公開され、指定したローカル暦月に属するものをcursor
-- より後から古い順に1ページ返す。published_atが同値の投稿はpost_idで
-- tie-breakされるため並び順は完全に決定的で、ページを順に辿ると各投稿を
-- ちょうど1回ずつ訪れる。
--
-- 1ページ目はゼロ時刻とゼロUUIDを渡す。どちらも保存されるどの行よりも前に
-- 並ぶため、1つの無条件な比較で1ページ目と2ページ目以降の両方をまかなえる。
-- cursorの有無フラグとORで包むと、パラメータ値がプラン時に未知の場合に
-- cursorを索引スキャンの開始位置として使えなくなる。
SELECT post_id, content, published_at FROM export_posts
WHERE export_id = sqlc.arg(export_id)
  AND published_at >= sqlc.arg(starts_at)
  AND published_at < sqlc.arg(ends_at)
  AND date_trunc('month', (published_at AT TIME ZONE 'UTC') AT TIME ZONE sqlc.arg(time_zone)::text) = sqlc.arg(local_month_start)::timestamp
  AND (published_at, post_id) > (sqlc.arg(after_published_at)::timestamp, sqlc.arg(after_id)::uuid)
ORDER BY published_at ASC, post_id ASC
LIMIT sqlc.arg(page_size);
