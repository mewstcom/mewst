package model_test

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

// exportInsert is a single attempted exports row. any-typed columns hold either
// nil (SQL NULL) or a concrete value so a test can exercise each branch of the
// exports_state_fields_check constraint.
//
// [Ja] exportInsert は 1 件の exports 挿入試行。any 型のカラムは nil (SQL NULL)
// か具体値を保持し、exports_state_fields_check の各分岐をテストで検証できる
// ようにする。
type exportInsert struct {
	profileID            model.ProfileID
	actorID              model.ActorID
	status               string
	objectKey            any
	attemptCount         int
	startedAt            any
	finishedAt           any
	completionNotifiedAt any
}

// attemptInsertExport inserts one exports row inside a savepoint so a constraint
// violation only rolls back this statement, letting the shared test transaction
// keep going for the next case. It returns the insert error (nil on success).
//
// [Ja] attemptInsertExport は 1 件の exports 行を savepoint 内で挿入し、制約
// 違反が起きてもこの文だけをロールバックして共有トランザクションを次のケースへ
// 継続させる。挿入エラー (成功時は nil) を返す。
func attemptInsertExport(t *testing.T, tx *sql.Tx, e exportInsert) error {
	t.Helper()

	if _, err := tx.Exec("SAVEPOINT sp_export"); err != nil {
		t.Fatalf("SAVEPOINT の作成に失敗: %v", err)
	}

	_, err := tx.Exec(`
		INSERT INTO exports (
			profile_id, actor_id, status, object_key, attempt_count,
			started_at, finished_at, completion_notified_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, uuid.UUID(e.profileID), uuid.UUID(e.actorID), e.status, e.objectKey,
		e.attemptCount, e.startedAt, e.finishedAt, e.completionNotifiedAt)

	if err != nil {
		if _, rbErr := tx.Exec("ROLLBACK TO SAVEPOINT sp_export"); rbErr != nil {
			t.Fatalf("ROLLBACK TO SAVEPOINT に失敗: %v", rbErr)
		}
		return err
	}
	if _, relErr := tx.Exec("RELEASE SAVEPOINT sp_export"); relErr != nil {
		t.Fatalf("RELEASE SAVEPOINT に失敗: %v", relErr)
	}
	return nil
}

// TestExportsSchemaConstraints verifies the DB-level invariants added by the
// create_exports migration: per-status field consistency, the value / range
// checks, the composite (actor_id, profile_id) foreign key with ON DELETE NO
// ACTION, and the partial unique index limiting active exports to one per
// profile. These invariants are exercised directly against a real database
// because PostgreSQL, rather than model methods, enforces them.
//
// [Ja] TestExportsSchemaConstraints は create_exports マイグレーションが追加する
// DB レベルの不変条件を検証する: status ごとのカラム整合性、値 / 範囲チェック、
// ON DELETE NO ACTION を持つ複合 (actor_id, profile_id) 外部キー、進行中
// エクスポートをプロフィールごとに 1 件へ制限する部分ユニークインデックス。
// これらの不変条件は model のメソッドではなく PostgreSQL が強制するため、実 DB に
// 対して直接検証する。
func TestExportsSchemaConstraints(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ts := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	const objectKey = "exports/profile/export.zip"

	userID := testutil.NewUserBuilder(t, tx).Build()

	// mkTarget creates a fresh profile plus an actor owned by userID for it, so
	// each sub-test gets its own profile and the partial unique index never
	// leaks state between cases.
	//
	// [Ja] mkTarget は新しいプロフィールと、それを対象とする userID 所有の actor を
	// 作る。各サブテストが自分のプロフィールを持ち、部分ユニークインデックスの状態が
	// ケース間で漏れないようにする。
	mkTarget := func(label string) (model.ProfileID, model.ActorID) {
		t.Helper()
		profileID := testutil.NewProfileBuilder(t, tx).
			WithAtname(fmt.Sprintf("exp_%s_%d", label, time.Now().UnixNano())).
			Build()
		actorID := testutil.NewActorBuilder(t, tx).
			WithUserID(userID).
			WithProfileID(profileID).
			Build()
		return profileID, actorID
	}

	t.Run("accepts a valid row for each status", func(t *testing.T) {
		cases := []struct {
			status                           string
			objectKey, startedAt, finishedAt any
		}{
			{"queued", nil, nil, nil},
			{"started", nil, ts, nil},
			{"succeeded", objectKey, ts, ts},
			{"failed", nil, ts, ts},
		}
		for _, c := range cases {
			profileID, actorID := mkTarget(c.status)
			if err := attemptInsertExport(t, tx, exportInsert{
				profileID: profileID, actorID: actorID, status: c.status,
				objectKey: c.objectKey, startedAt: c.startedAt, finishedAt: c.finishedAt,
			}); err != nil {
				t.Errorf("status=%q の有効な行が拒否された: %v", c.status, err)
			}
		}
	})

	t.Run("rejects state field combinations that contradict the status", func(t *testing.T) {
		cases := []struct {
			name string
			row  exportInsert
		}{
			{"queued with started_at", exportInsert{status: "queued", startedAt: ts}},
			{"started without started_at", exportInsert{status: "started"}},
			{"succeeded without object_key", exportInsert{status: "succeeded", startedAt: ts, finishedAt: ts}},
			{"failed with object_key", exportInsert{status: "failed", objectKey: objectKey, startedAt: ts, finishedAt: ts}},
		}
		for _, c := range cases {
			profileID, actorID := mkTarget("state")
			row := c.row
			row.profileID, row.actorID = profileID, actorID
			err := attemptInsertExport(t, tx, row)
			if err == nil {
				t.Errorf("%s: 拒否されるべき行が挿入できた", c.name)
				continue
			}
			if !strings.Contains(err.Error(), "exports_state_fields_check") {
				t.Errorf("%s: exports_state_fields_check 違反を期待したが別のエラー: %v", c.name, err)
			}
		}
	})

	t.Run("rejects an unknown status", func(t *testing.T) {
		profileID, actorID := mkTarget("status")
		err := attemptInsertExport(t, tx, exportInsert{
			profileID: profileID, actorID: actorID, status: "cancelled",
		})
		// A status outside the enum has no matching branch in either check, so
		// PostgreSQL may report whichever it evaluates first. Accepting both
		// keeps the test about "an unknown status cannot be stored" rather than
		// which redundant guard fires.
		//
		// [Ja] enum 外の status はどちらの check にも一致する分岐がないため、
		// PostgreSQL は先に評価した方を報告しうる。両方を許容し、「未知の status を
		// 保存できない」ことを検証する意図を保つ (どちらの冗長なガードが発火するかは
		// 問わない)。
		if err == nil ||
			(!strings.Contains(err.Error(), "exports_status_check") &&
				!strings.Contains(err.Error(), "exports_state_fields_check")) {
			t.Errorf("status の check 制約違反を期待したが: %v", err)
		}
	})

	t.Run("rejects a negative attempt_count", func(t *testing.T) {
		profileID, actorID := mkTarget("attempt")
		err := attemptInsertExport(t, tx, exportInsert{
			profileID: profileID, actorID: actorID, status: "queued", attemptCount: -1,
		})
		if err == nil || !strings.Contains(err.Error(), "exports_attempt_count_check") {
			t.Errorf("exports_attempt_count_check 違反を期待したが: %v", err)
		}
	})

	t.Run("rejects completion_notified_at unless succeeded", func(t *testing.T) {
		profileID, actorID := mkTarget("notified")
		err := attemptInsertExport(t, tx, exportInsert{
			profileID: profileID, actorID: actorID, status: "started",
			startedAt: ts, completionNotifiedAt: ts,
		})
		if err == nil || !strings.Contains(err.Error(), "exports_completion_notified_at_check") {
			t.Errorf("exports_completion_notified_at_check 違反を期待したが: %v", err)
		}
	})

	t.Run("rejects an actor whose profile differs from the export target", func(t *testing.T) {
		_, actorA := mkTarget("mismatchA")
		profileB, _ := mkTarget("mismatchB")
		// actorA belongs to profile A, so pairing it with profile B must be
		// rejected by the composite foreign key.
		//
		// [Ja] actorA はプロフィール A に属するため、プロフィール B と組み合わせると
		// 複合外部キーに拒否される。
		err := attemptInsertExport(t, tx, exportInsert{
			profileID: profileB, actorID: actorA, status: "queued",
		})
		if err == nil || !strings.Contains(err.Error(), "exports_actor_profile_fkey") {
			t.Errorf("exports_actor_profile_fkey 違反を期待したが: %v", err)
		}
	})

	t.Run("allows only one active export per profile", func(t *testing.T) {
		profileID, actorID := mkTarget("active")

		if err := attemptInsertExport(t, tx, exportInsert{
			profileID: profileID, actorID: actorID, status: "queued",
		}); err != nil {
			t.Fatalf("最初の queued の挿入に失敗: %v", err)
		}

		// A second active (started) export for the same profile must hit the
		// partial unique index.
		//
		// [Ja] 同一プロフィールの 2 件目の active (started) エクスポートは部分
		// ユニークインデックスに衝突する。
		err := attemptInsertExport(t, tx, exportInsert{
			profileID: profileID, actorID: actorID, status: "started", startedAt: ts,
		})
		if err == nil || !strings.Contains(err.Error(), "index_exports_on_profile_id_where_active") {
			t.Errorf("index_exports_on_profile_id_where_active 違反を期待したが: %v", err)
		}

		// succeeded / failed rows are outside the partial index, so multiple of
		// them may coexist with the active row.
		//
		// [Ja] succeeded / failed の行は部分インデックスの対象外なので、active な行と
		// 複数共存できる。
		for i := 0; i < 2; i++ {
			if err := attemptInsertExport(t, tx, exportInsert{
				profileID: profileID, actorID: actorID, status: "succeeded",
				objectKey: objectKey, startedAt: ts, finishedAt: ts,
			}); err != nil {
				t.Errorf("succeeded の共存を期待したが %d 件目で失敗: %v", i+1, err)
			}
		}
	})

	t.Run("creates the lookup and grouping indexes", func(t *testing.T) {
		// These are performance indexes, so no insert can observe them; assert
		// their presence against the catalog instead.
		// index_exports_on_profile_id_where_active is exercised behaviorally above,
		// and the recovery indexes are checked for their key order below.
		//
		// [Ja] これらは性能用インデックスで挿入では観測できないため、代わりに
		// カタログで存在を確認する。index_exports_on_profile_id_where_active は上の
		// テストで挙動として検証しており、回復用インデックスは下でキー順を確認する。
		want := []string{
			"index_exports_on_profile_id_and_created_at",
			"index_exports_on_actor_id_and_profile_id",
			"index_exports_on_profile_id_where_succeeded",
		}
		for _, name := range want {
			var found bool
			if err := tx.QueryRow(`
				SELECT EXISTS (
					SELECT 1 FROM pg_indexes
					WHERE schemaname = 'public' AND tablename = 'exports' AND indexname = $1
				)
			`, name).Scan(&found); err != nil {
				t.Fatalf("%s の存在確認クエリに失敗: %v", name, err)
			}
			if !found {
				t.Errorf("インデックス %s が存在しない", name)
			}
		}
	})

	t.Run("creates recovery indexes with the full keyset order", func(t *testing.T) {
		// Each recovery cursor compares the timestamp and id as a row value. Check
		// the catalog definition so the indexes keep supporting that complete order.
		//
		// [Ja] 各回復 cursor は時刻と id を行値として比較する。完全な並び順を
		// インデックスが支え続けるよう、カタログ上の定義を確認する。
		want := map[string]string{
			"index_exports_on_created_at_where_queued":      "(created_at, id)",
			"index_exports_on_started_at_where_started":     "(started_at, id)",
			"index_exports_on_finished_at_where_unnotified": "(finished_at, id)",
		}
		for name, columns := range want {
			var indexDef string
			if err := tx.QueryRow(`
				SELECT indexdef FROM pg_indexes
				WHERE schemaname = 'public' AND tablename = 'exports' AND indexname = $1
			`, name).Scan(&indexDef); err != nil {
				t.Fatalf("%s の定義取得クエリに失敗: %v", name, err)
			}
			if !strings.Contains(indexDef, columns) {
				t.Errorf("インデックス %s のキーが不正: got %q, want to contain %q", name, indexDef, columns)
			}
		}
	})

	t.Run("blocks deleting an actor still referenced by an export", func(t *testing.T) {
		profileID, actorID := mkTarget("noaction")
		if err := attemptInsertExport(t, tx, exportInsert{
			profileID: profileID, actorID: actorID, status: "succeeded",
			objectKey: objectKey, startedAt: ts, finishedAt: ts,
		}); err != nil {
			t.Fatalf("succeeded の挿入に失敗: %v", err)
		}

		if _, err := tx.Exec("SAVEPOINT sp_delete"); err != nil {
			t.Fatalf("SAVEPOINT の作成に失敗: %v", err)
		}
		_, err := tx.Exec("DELETE FROM actors WHERE id = $1", uuid.UUID(actorID))
		if _, rbErr := tx.Exec("ROLLBACK TO SAVEPOINT sp_delete"); rbErr != nil {
			t.Fatalf("ROLLBACK TO SAVEPOINT に失敗: %v", rbErr)
		}
		// ON DELETE NO ACTION means the referenced actor cannot be removed while
		// an export still points at it; a CASCADE would have deleted it instead.
		//
		// [Ja] ON DELETE NO ACTION により、エクスポートが参照している間は actor を
		// 削除できない。CASCADE なら代わりに削除されてしまう。
		if err == nil || !strings.Contains(err.Error(), "exports_actor_profile_fkey") {
			t.Errorf("exports_actor_profile_fkey による削除拒否を期待したが: %v", err)
		}
	})
}
