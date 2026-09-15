-- migrate:up

-- エクスポート申請時に見えていた投稿を固定化する。post_idは意図的に外部キーに
-- しない。Railsはdiscardの直後に元投稿を物理削除し得る一方、処理中の
-- エクスポートは申請時点の本文を読み続ける必要がある。これらの行はexport専用の
-- スナップショットなので、export削除時は安全にcascade削除できる。
--
-- contentとpublished_atは、Go版のカラム定義ガイドラインが求めるVARCHAR /
-- TIMESTAMP WITH TIME ZONEではなく、複製元のpostsカラムと同じ型 (textと、
-- UTCの壁時計を保持するtimestamp without time zone) にする。複製元と型を揃えると
-- 変換を挟まずに済み、timestamptzにするとtimestampからの暗黙キャストが
-- セッションのTimeZone設定で解決されるため、AT TIME ZONE 'UTC' を1箇所
-- 書き忘れるだけでpublished_atが静かにずれる。
--
-- 主キーはアーカイブのページング用インデックスを兼ねる。各月は
-- (published_at, post_id) 順のkeyset cursorで走査するため、export_idを先頭に
-- することで各ページを1つの不変なsnapshotに限定し、完全なcursorを
-- インデックス走査の開始位置にできる。1本のキーで両方の役割を担わせることで、
-- 投稿数の多いプロフィールの申請時複製が同じ行に対して2本目のB-treeを
-- 維持せずに済む。キーがpost_idよりpublished_atを先に並べるため、一意性が
-- 効くのは (export_id, post_id) ではなく (export_id, published_at, post_id) に
-- なるが、複製はpostsに対する単一のINSERT ... SELECTなので、1つのexportに
-- 同じ投稿が2回入ることはない。
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
