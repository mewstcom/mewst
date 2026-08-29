package seed

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/lib/pq"

	"github.com/mewstcom/mewst/go/internal/testutil"
)

// TestCleanupTablesAndPreservedTablesCoverTheSchema verifies that every table
// in the schema is named in one of the two lists, and in only one of them.
//
// A table added later is what this is for. Without the check it would simply
// not be emptied, and the run after it was added would leave its rows in place
// while every screen around them was rebuilt, which is the state the seed
// exists to make impossible.
//
// [Ja] TestCleanupTablesAndPreservedTablesCoverTheSchema は、スキーマのすべての
// テーブルが 2 つの一覧のどちらかに、かつ一方だけに挙げられていることを検証する。
//
// これが備えているのは、後から追加されるテーブルである。この検査が無いと、その
// テーブルは単に空にされず、追加後の実行は、周囲のすべての画面が作り直される
// かたわらでその行を残す。それはシードが不可能にするために存在する状態そのもの。
func TestCleanupTablesAndPreservedTablesCoverTheSchema(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	ctx := context.Background()

	rows, err := db.QueryContext(ctx, `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`)
	if err != nil {
		t.Fatalf("スキーマのテーブル一覧の取得に失敗: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var schemaTables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			t.Fatalf("テーブル名の読み取りに失敗: %v", err)
		}
		schemaTables = append(schemaTables, table)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("スキーマのテーブル一覧の読み取りに失敗: %v", err)
	}
	if len(schemaTables) == 0 {
		t.Fatal("スキーマにテーブルが 1 つもない。テスト DB にスキーマが適用されているか確認すること")
	}

	listed := make(map[string]int, len(cleanupTables)+len(preservedTables))
	for _, table := range cleanupTables {
		listed[table]++
	}
	for _, table := range preservedTables {
		listed[table]++
	}

	for _, table := range schemaTables {
		switch listed[table] {
		case 0:
			t.Errorf(
				"テーブル %s がクリーンアップの対象にも対象外にも挙げられていない。cleanupTables か preservedTables のどちらかへ追加すること",
				table,
			)
		case 1:
		default:
			t.Errorf("テーブル %s が cleanupTables と preservedTables の両方に挙げられている", table)
		}
	}

	// A table named in a list but gone from the schema is the other half of
	// the same drift: it makes the cleanup statement fail on a database that
	// no longer has it.
	//
	// [Ja] 一覧に挙げられているのにスキーマから消えているテーブルは、同じずれの
	// もう半分である。そのテーブルを持たなくなったデータベースで、クリーンアップの
	// 文を失敗させることになる。
	for table := range listed {
		if !slices.Contains(schemaTables, table) {
			t.Errorf("テーブル %s が一覧に挙げられているがスキーマに存在しない", table)
		}
	}
}

// TestPreservedTablesAreOutOfCascadeReach verifies that no preserved table
// references a table the cleanup empties.
//
// TRUNCATE ... CASCADE empties every table that references one named in the
// statement, whether or not it was named. A foreign key added later from a job
// queue table to an application table would therefore hand the cleanup a
// preserved table to empty, and it would do so without a word.
//
// [Ja] TestPreservedTablesAreOutOfCascadeReach は、クリーンアップが空にする
// テーブルを参照する対象外のテーブルが無いことを検証する。
//
// TRUNCATE ... CASCADE は、文に挙げられたテーブルを参照するテーブルを、それが
// 文に挙げられているかどうかによらず空にする。後からジョブキューのテーブルから
// アプリケーションのテーブルへ外部キーが張られると、クリーンアップは対象外の
// テーブルを空にすることになり、しかもそれを何も告げずに行う。
func TestPreservedTablesAreOutOfCascadeReach(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	ctx := context.Background()

	rows, err := db.QueryContext(ctx, `
		SELECT referencing.relname, referenced.relname
		FROM pg_constraint AS c
		JOIN pg_class AS referencing ON referencing.oid = c.conrelid
		JOIN pg_class AS referenced ON referenced.oid = c.confrelid
		WHERE c.contype = 'f'
		  AND referencing.relname = ANY($1)
		  AND referenced.relname = ANY($2)
	`, pq.Array(preservedTables), pq.Array(cleanupTables))
	if err != nil {
		t.Fatalf("外部キーの取得に失敗: %v", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var referencing, referenced string
		if err := rows.Scan(&referencing, &referenced); err != nil {
			t.Fatalf("外部キーの読み取りに失敗: %v", err)
		}
		t.Errorf(
			"対象外のテーブル %s が対象の %s を参照している。TRUNCATE ... CASCADE がこれを空にしてしまうため、どちらの一覧に置くかを見直すこと",
			referencing, referenced,
		)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("外部キーの読み取りに失敗: %v", err)
	}
}

// TestCleanupStatement verifies that the statement empties exactly the tables
// the list names, with each identifier quoted.
//
// [Ja] TestCleanupStatement は、文が一覧の挙げるテーブルをちょうど空にすること、
// 識別子がそれぞれクォートされていることを検証する。
func TestCleanupStatement(t *testing.T) {
	t.Parallel()

	statement := cleanupStatement()

	if !strings.HasPrefix(statement, "TRUNCATE TABLE ") {
		t.Errorf("文が TRUNCATE で始まることを期待したが %q だった", statement)
	}

	// CASCADE is what lets tables that reference each other be emptied in one
	// statement. Without it the cleanup is rejected by the first foreign key
	// it meets.
	//
	// [Ja] CASCADE は、互いを参照し合うテーブルを 1 文で空にすることを可能にする
	// もの。これが無いと、クリーンアップは最初に出会った外部キーによって拒否される。
	if !strings.HasSuffix(statement, " CASCADE") {
		t.Errorf("文が CASCADE で終わることを期待したが %q だった", statement)
	}

	names := strings.Split(strings.TrimSuffix(strings.TrimPrefix(statement, "TRUNCATE TABLE "), " CASCADE"), ", ")
	if len(names) != len(cleanupTables) {
		t.Fatalf("文が %d 個のテーブルを挙げることを期待したが %d 個だった: %q", len(cleanupTables), len(names), statement)
	}
	for i, name := range names {
		if want := `"` + cleanupTables[i] + `"`; name != want {
			t.Errorf("%d 番目のテーブルが %s であることを期待したが %s だった", i+1, want, name)
		}
	}
}
