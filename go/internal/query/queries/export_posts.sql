-- name: ListExportPostMonthsByExportID :many
-- Return one row per calendar month in an export's immutable post snapshot,
-- with the post count and a UTC scan range containing those posts, oldest
-- first. The export writes one HTML file per month and needs the counts before
-- the first file, so the summary is taken up front and each month is then paged
-- over its range from the same materialized snapshot.
--
-- export_posts.published_at is timestamp without time zone holding UTC, so the
-- month a post belongs to is found by reading it as UTC and converting to the
-- target zone before truncating. A local month start can be ambiguous during a
-- daylight saving fold, so converting that wall clock back to one UTC instant
-- is not a safe scan boundary. Instead, MIN / MAX derive a tight half-open
-- range from the exact export_posts rows in the group. The paging query repeats
-- the local-month predicate as a correctness guard and uses the range for its
-- index scan.
--
-- [Ja] export の不変な投稿 snapshot に含まれる暦月ごとに 1 行を、投稿件数と
-- その投稿を含む UTC 走査範囲とともに古い順で返す。
-- エクスポートは月ごとに 1 つの HTML ファイルを書き、最初のファイルより前に
-- 件数を必要とするため、先にサマリーを取得し、その後で同じ固定済み snapshot
-- から各月をその範囲で分割取得する。
--
-- export_posts.published_at は UTC を保持する timestamp without time zone
-- のため、投稿が属する月は UTC として読んでから対象タイムゾーンへ変換し、
-- truncate して求める。夏時間のフォールド中はローカル月初が曖昧になりうるため、
-- その壁時計を 1 つの UTC 時刻へ逆変換しても安全な走査境界にはならない。
-- 代わりに、グループの正確な export_posts 行から MIN / MAX で狭い半開区間を
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
-- Return a page from an export's immutable post snapshot, published within the
-- half-open UTC scan range [starts_at, ends_at), and in the requested local
-- calendar month. Rows are oldest first and strictly after the cursor. The
-- order is fully deterministic because post_id breaks ties between posts
-- sharing one published_at, so successive pages visit every post exactly once.
--
-- The first page passes the zero timestamp and the zero UUID, which sort before
-- every stored row, so one unconditional comparison drives both the first and
-- later pages. Wrapping it in an OR with a has-cursor flag instead would keep
-- the planner from using the cursor as the starting point of an index scan
-- whenever the parameter value is unknown at plan time.
--
-- [Ja] export の不変な投稿 snapshot のうち、半開区間の UTC 走査範囲
-- [starts_at, ends_at) に公開され、指定したローカル暦月に属するものを cursor
-- より後から古い順に 1 ページ返す。published_at が同値の投稿は post_id で
-- tie-break されるため並び順は完全に決定的で、ページを順に辿ると各投稿を
-- ちょうど 1 回ずつ訪れる。
--
-- 1 ページ目はゼロ時刻とゼロ UUID を渡す。どちらも保存されるどの行よりも前に
-- 並ぶため、1 つの無条件な比較で 1 ページ目と 2 ページ目以降の両方をまかなえる。
-- cursor の有無フラグと OR で包むと、パラメータ値がプラン時に未知の場合に
-- cursor を索引スキャンの開始位置として使えなくなる。
SELECT post_id, content, published_at FROM export_posts
WHERE export_id = sqlc.arg(export_id)
  AND published_at >= sqlc.arg(starts_at)
  AND published_at < sqlc.arg(ends_at)
  AND date_trunc('month', (published_at AT TIME ZONE 'UTC') AT TIME ZONE sqlc.arg(time_zone)::text) = sqlc.arg(local_month_start)::timestamp
  AND (published_at, post_id) > (sqlc.arg(after_published_at)::timestamp, sqlc.arg(after_id)::uuid)
ORDER BY published_at ASC, post_id ASC
LIMIT sqlc.arg(page_size);
