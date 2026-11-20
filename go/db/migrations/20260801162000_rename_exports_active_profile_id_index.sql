-- migrate:up

-- activeなエクスポートに対する部分ユニークインデックスを、このテーブルの他の
-- 部分インデックスが使う _where_{述語} 形式へ改名する。同じ種類のインデックスに
-- 2つの命名の流儀が並ばないようにするため。
ALTER INDEX index_exports_on_active_profile_id
    RENAME TO index_exports_on_profile_id_where_active;

-- migrate:down

ALTER INDEX index_exports_on_profile_id_where_active
    RENAME TO index_exports_on_active_profile_id;
