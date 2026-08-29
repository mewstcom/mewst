-- migrate:up

-- Materialize the posts visible when an export is requested. post_id is
-- deliberately not a foreign key: Rails may physically delete the source post
-- immediately after discard, while an in-progress export must keep reading the
-- request-time content. The export owns these rows, so deleting an export can
-- safely cascade to its private snapshot.
--
-- content and published_at repeat the types of the posts columns they copy
-- (text, and a timestamp without time zone holding a UTC wall clock) instead of
-- the VARCHAR / TIMESTAMP WITH TIME ZONE the Go column guidelines ask for.
-- Matching the source keeps the copy free of conversions, and a timestamptz
-- target would resolve the implicit cast through the session TimeZone, so one
-- missing AT TIME ZONE 'UTC' would silently shift every published_at.
--
-- The primary key doubles as the archive's paging index. Each month is walked
-- in (published_at, post_id) order with a keyset cursor, so leading with
-- export_id scopes every page to one immutable snapshot and lets the full
-- cursor become the index scan's starting point. Serving both roles from one
-- key spares the request-time copy of a large profile a second B-tree over the
-- same rows. Ordering the key by published_at before post_id means uniqueness
-- covers (export_id, published_at, post_id) rather than (export_id, post_id):
-- the copy is a single INSERT ... SELECT over posts, so one export cannot
-- contain the same post twice.
--
-- [Ja] エクスポート申請時に見えていた投稿を固定化する。post_id は意図的に外部キーに
-- しない。Rails は discard の直後に元投稿を物理削除し得る一方、処理中の
-- エクスポートは申請時点の本文を読み続ける必要がある。これらの行は export 専用の
-- スナップショットなので、export 削除時は安全に cascade 削除できる。
--
-- content と published_at は、Go 版のカラム定義ガイドラインが求める VARCHAR /
-- TIMESTAMP WITH TIME ZONE ではなく、複製元の posts カラムと同じ型 (text と、
-- UTC の壁時計を保持する timestamp without time zone) にする。複製元と型を揃えると
-- 変換を挟まずに済み、timestamptz にすると timestamp からの暗黙キャストが
-- セッションの TimeZone 設定で解決されるため、AT TIME ZONE 'UTC' を 1 箇所
-- 書き忘れるだけで published_at が静かにずれる。
--
-- 主キーはアーカイブのページング用インデックスを兼ねる。各月は
-- (published_at, post_id) 順の keyset cursor で走査するため、export_id を先頭に
-- することで各ページを 1 つの不変な snapshot に限定し、完全な cursor を
-- インデックス走査の開始位置にできる。1 本のキーで両方の役割を担わせることで、
-- 投稿数の多いプロフィールの申請時複製が同じ行に対して 2 本目の B-tree を
-- 維持せずに済む。キーが post_id より published_at を先に並べるため、一意性が
-- 効くのは (export_id, post_id) ではなく (export_id, published_at, post_id) に
-- なるが、複製は posts に対する単一の INSERT ... SELECT なので、1 つの export に
-- 同じ投稿が 2 回入ることはない。
CREATE TABLE export_posts (
    export_id uuid NOT NULL
        REFERENCES exports (id) ON DELETE CASCADE,
    post_id uuid NOT NULL,
    content text NOT NULL,
    published_at timestamp without time zone NOT NULL,
    PRIMARY KEY (export_id, published_at, post_id)
);

-- migrate:down

DROP TABLE IF EXISTS export_posts;
