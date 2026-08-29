package seed

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"
)

// cleanupTables are the tables a run empties before it generates. Between them
// and preservedTables, every table in the schema is accounted for, which is
// what keeps a table added later from quietly falling out of the cleanup: it
// has to be put in one list or the other.
//
// oauth_applications is emptied and written again by the seed rather than left
// to Rails. posts.oauth_application_id is NOT NULL and every generated post
// points at that one row, so a seed that does not create it can only run after
// bin/rails db:seed has, in that order, on a database nobody has emptied since.
//
// [Ja] cleanupTables は、実行が生成の前に空にするテーブル。これと
// preservedTables を合わせるとスキーマのすべてのテーブルが揃う。後から追加された
// テーブルが黙ってクリーンアップから漏れることを防ぐためであり、どちらか一方の
// 一覧へ必ず入れることになる。
//
// oauth_applications は Rails に任せず、シードが空にして入れ直す。
// posts.oauth_application_id は NOT NULL で、生成されるすべてのポストがその 1 行を
// 指す。その行を作らないシードは、bin/rails db:seed をその順序で先に実行し、以降
// 誰もデータベースを空にしていないという条件のもとでしか実行できない。
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

// preservedTables are the tables a run leaves alone. None of them holds
// application data: they record which migrations have been applied, or they
// are the bookkeeping of a job queue. Emptying those would not reset a screen,
// it would tell the two frameworks that their schema and their queues are
// something other than what they are.
//
// [Ja] preservedTables は、実行が触れないテーブル。いずれもアプリケーションの
// データを持たない。適用済みのマイグレーションを記録するものか、ジョブキューの
// 管理情報である。これらを空にしても画面が初期化されることはなく、2 つの
// フレームワークに対して、スキーマとキューが実際とは違うものであると告げることに
// なる。
var preservedTables = []string{
	// Migration bookkeeping.
	//
	// [Ja] マイグレーションの管理情報。
	"ar_internal_metadata",
	"schema_migrations",

	// The Go version's job queue (River).
	//
	// [Ja] Go 版のジョブキュー (River)。
	"river_job",
	"river_leader",
	"river_migration",
	"river_notification",
	"river_queue",

	// The Rails version's job queue (GoodJob).
	//
	// [Ja] Rails 版のジョブキュー (GoodJob)。
	"good_job_batches",
	"good_job_executions",
	"good_job_processes",
	"good_job_settings",
	"good_jobs",
}

// cleanup empties every table in cleanupTables.
//
// TRUNCATE takes the tables in one statement, which is what lets it empty
// tables that reference each other without an order to do it in. CASCADE is
// what makes that legal: a table left out of the statement but referencing one
// in it would otherwise be rejected. No preserved table references a cleanup
// table, so CASCADE reaches nothing beyond the list, which is what
// TestPreservedTablesAreOutOfCascadeReach holds it to.
//
// [Ja] cleanup は cleanupTables のすべてのテーブルを空にする。
//
// TRUNCATE は複数のテーブルを 1 文で受け取る。それにより、互いを参照し合う
// テーブルを、順序を決めることなく空にできる。CASCADE はそれを可能にするもので、
// 文に挙げられていないのに挙げられたテーブルを参照するテーブルがあると、そうしない
// かぎり拒否される。クリーンアップ対象を参照する対象外のテーブルは無いため、
// CASCADE が一覧の外へ及ぶことはない。それを固定するのが
// TestPreservedTablesAreOutOfCascadeReach である。
func cleanup(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, cleanupStatement()); err != nil {
		return fmt.Errorf("既存データの削除に失敗: %w", err)
	}

	return nil
}

// cleanupStatement builds the statement cleanup runs. It is separate so that
// what a run is about to execute can be read in a test without a database
// having to be emptied to see it.
//
// [Ja] cleanupStatement は cleanup が実行する文を組み立てる。実行がこれから何を
// 実行するのかを、それを見るためにデータベースを空にすることなくテストから読める
// ようにするため、分けている。
func cleanupStatement() string {
	quoted := make([]string, 0, len(cleanupTables))
	for _, table := range cleanupTables {
		quoted = append(quoted, pq.QuoteIdentifier(table))
	}

	// The statement is built rather than written out because the list above is
	// what the completeness test checks against the schema, and a second copy
	// of it written into a string literal would be the copy that drifts.
	//
	// Nothing from outside the program reaches this string: the only values
	// interpolated are the identifiers of the constant list above, each put
	// through the quoting the driver provides.
	//
	// [Ja] 文をベタ書きせず組み立てるのは、網羅性のテストがスキーマと突き合わせる
	// のが上記の一覧であり、それを文字列リテラルへ書き写した 2 つ目の写しこそが
	// ずれていく側になるため。
	//
	// この文字列にプログラムの外から届くものは無い。埋め込まれるのは上記の定数の
	// 一覧の識別子だけであり、それぞれをドライバが提供するクォートに通している。
	return "TRUNCATE TABLE " + strings.Join(quoted, ", ") + " CASCADE" // #nosec G202
}
