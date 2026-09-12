-- migrate:up

-- The export feature is released to every user, so its flag records no longer
-- gate anything and the application no longer defines the name.
--
-- [Ja] エクスポート機能は全ユーザーに公開され、そのフラグレコードはもう何も
-- ゲートしておらず、アプリケーションもこの名前を定義しなくなった。
DELETE FROM feature_flags WHERE name = 'go_export';

-- migrate:down

-- Down cannot restore who held the flag: which actors and devices were granted
-- it is not recorded anywhere else. Rolling back leaves the table without the
-- rows, which is the same state a fresh grant starts from.
--
-- [Ja] 誰がフラグを持っていたかは他のどこにも記録が無いため、down では復元でき
-- ない。ロールバックしてもテーブルには行が無い状態が残るが、これは新たに付与を
-- やり直すときの出発点と同じ状態である。
