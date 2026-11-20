package seed

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"
)

// cleanupTablesは、実行が生成の前に空にするテーブル。これと
// preservedTablesを合わせるとスキーマのすべてのテーブルが揃う。後から追加された
// テーブルが黙ってクリーンアップから漏れることを防ぐためであり、どちらか一方の
// 一覧へ必ず入れることになる。
//
// oauth_applicationsはシードが空にして入れ直す。
// posts.oauth_application_idはNOT NULLで、生成されるすべてのポストがその行を
// 指す。クリーンアップ対象に含めることで、生成するポストが依存するアプリケーション
// データをシード自身がすべて所有する。
var cleanupTables = []string{
	"actors",
	"email_confirmations",
	"export_completion_notifications",
	"export_posts",
	"exports",
	"feature_flags",
	"follows",
	"home_timeline_posts",
	"links",
	"notifications",
	"oauth_access_grants",
	"oauth_access_tokens",
	"oauth_applications",
	"post_links",
	"posts",
	"profiles",
	"rate_limits",
	"sessions",
	"stamps",
	"suggested_follows",
	"user_profiles",
	"users",
}

// preservedTablesは、実行が触れないテーブル。いずれもアプリケーションの
// データを持たない。適用済みのマイグレーションを記録するものか、ジョブキューの
// 管理情報である。これらを空にしても画面が初期化されることはなく、2つの
// フレームワークに対して、スキーマとキューが実際とは違うものであると告げることに
// なる。
var preservedTables = []string{
	// マイグレーションの管理情報。
	"ar_internal_metadata",
	"schema_migrations",

	// Go版のジョブキュー (River)。
	"river_job",
	"river_leader",
	"river_migration",
	"river_notification",
	"river_queue",

	// Rails版のジョブキュー (GoodJob)。
	"good_job_batches",
	"good_job_executions",
	"good_job_processes",
	"good_job_settings",
	"good_jobs",
}

// cleanupはcleanupTablesのすべてのテーブルを空にする。
//
// TRUNCATEは複数のテーブルを1文で受け取る。それにより、互いを参照し合う
// テーブルを、順序を決めることなく空にできる。CASCADEはそれを可能にするもので、
// 文に挙げられていないのに挙げられたテーブルを参照するテーブルがあると、そうしない
// かぎり拒否される。クリーンアップ対象を参照する対象外のテーブルは無いため、
// CASCADEが一覧の外へ及ぶことはない。それを固定するのが
// TestPreservedTablesAreOutOfCascadeReachである。
func cleanup(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, cleanupStatement()); err != nil {
		return fmt.Errorf("既存データの削除に失敗: %w", err)
	}

	return nil
}

// cleanupStatementはcleanupが実行する文を組み立てる。実行がこれから何を
// 実行するのかを、それを見るためにデータベースを空にすることなくテストから読める
// ようにするため、分けている。
func cleanupStatement() string {
	quoted := make([]string, 0, len(cleanupTables))
	for _, table := range cleanupTables {
		quoted = append(quoted, pq.QuoteIdentifier(table))
	}

	// 文をベタ書きせず組み立てるのは、網羅性のテストがスキーマと突き合わせる
	// のが上記の一覧であり、それを文字列リテラルへ書き写した2つ目の写しこそが
	// ずれていく側になるため。
	//
	// この文字列にプログラムの外から届くものは無い。埋め込まれるのは上記の定数の
	// 一覧の識別子だけであり、それぞれをドライバが提供するクォートに通している。
	return "TRUNCATE TABLE " + strings.Join(quoted, ", ") + " CASCADE" // #nosec G202
}
