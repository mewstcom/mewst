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

// exportInsertは1件のexports挿入試行。any型のカラムはnil (SQL NULL)
// か具体値を保持し、exports_state_fields_checkの各分岐をテストで検証できる
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

// attemptInsertExportは1件のexports行をsavepoint内で挿入し、制約
// 違反が起きてもこの文だけをロールバックして共有トランザクションを次のケースへ
// 継続させる。挿入エラー (成功時はnil) を返す。
func attemptInsertExport(t *testing.T, tx *sql.Tx, e exportInsert) error {
	t.Helper()

	if _, err := tx.Exec("SAVEPOINT sp_export"); err != nil {
		t.Fatalf("SAVEPOINTの作成に失敗: %v", err)
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
			t.Fatalf("ROLLBACK TO SAVEPOINTに失敗: %v", rbErr)
		}
		return err
	}
	if _, relErr := tx.Exec("RELEASE SAVEPOINT sp_export"); relErr != nil {
		t.Fatalf("RELEASE SAVEPOINTに失敗: %v", relErr)
	}
	return nil
}

// TestExportsSchemaConstraintsはcreate_exportsマイグレーションが追加する
// DBレベルの不変条件を検証する: statusごとのカラム整合性、値 / 範囲チェック、
// ON DELETE NO ACTIONを持つ複合 (actor_id, profile_id) 外部キー、進行中
// エクスポートをプロフィールごとに1件へ制限する部分ユニークインデックス。
// これらの不変条件はmodelのメソッドではなくPostgreSQLが強制するため、実DBに
// 対して直接検証する。
func TestExportsSchemaConstraints(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ts := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	const objectKey = "exports/profile/export.zip"

	userID := testutil.NewUserBuilder(t, tx).Build()

	// mkTargetは新しいプロフィールと、それを対象とするuserID所有のactorを
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

	t.Run("statusごとの妥当な行を受け入れる", func(t *testing.T) {
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
				t.Errorf("status=%qの有効な行が拒否された: %v", c.status, err)
			}
		}
	})

	t.Run("statusと矛盾する状態フィールドの組み合わせを拒否する", func(t *testing.T) {
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
				t.Errorf("%s: exports_state_fields_check違反を期待したが別のエラー: %v", c.name, err)
			}
		}
	})

	t.Run("未知のstatusを拒否する", func(t *testing.T) {
		profileID, actorID := mkTarget("status")
		err := attemptInsertExport(t, tx, exportInsert{
			profileID: profileID, actorID: actorID, status: "cancelled",
		})
		// enum外のstatusはどちらのcheckにも一致する分岐がないため、
		// PostgreSQLは先に評価した方を報告しうる。両方を許容し、「未知のstatusを
		// 保存できない」ことを検証する意図を保つ (どちらの冗長なガードが発火するかは
		// 問わない)。
		if err == nil ||
			(!strings.Contains(err.Error(), "exports_status_check") &&
				!strings.Contains(err.Error(), "exports_state_fields_check")) {
			t.Errorf("statusのcheck制約違反を期待したが: %v", err)
		}
	})

	t.Run("負のattempt_countを拒否する", func(t *testing.T) {
		profileID, actorID := mkTarget("attempt")
		err := attemptInsertExport(t, tx, exportInsert{
			profileID: profileID, actorID: actorID, status: "queued", attemptCount: -1,
		})
		if err == nil || !strings.Contains(err.Error(), "exports_attempt_count_check") {
			t.Errorf("exports_attempt_count_check違反を期待したが: %v", err)
		}
	})

	t.Run("succeeded以外でのcompletion_notified_atを拒否する", func(t *testing.T) {
		profileID, actorID := mkTarget("notified")
		err := attemptInsertExport(t, tx, exportInsert{
			profileID: profileID, actorID: actorID, status: "started",
			startedAt: ts, completionNotifiedAt: ts,
		})
		if err == nil || !strings.Contains(err.Error(), "exports_completion_notified_at_check") {
			t.Errorf("exports_completion_notified_at_check違反を期待したが: %v", err)
		}
	})

	t.Run("エクスポート対象と別のプロフィールに属するactorを拒否する", func(t *testing.T) {
		_, actorA := mkTarget("mismatchA")
		profileB, _ := mkTarget("mismatchB")
		// actorAはプロフィールAに属するため、プロフィールBと組み合わせると
		// 複合外部キーに拒否される。
		err := attemptInsertExport(t, tx, exportInsert{
			profileID: profileB, actorID: actorA, status: "queued",
		})
		if err == nil || !strings.Contains(err.Error(), "exports_actor_profile_fkey") {
			t.Errorf("exports_actor_profile_fkey違反を期待したが: %v", err)
		}
	})

	t.Run("プロフィールごとに進行中のエクスポートを1件だけ許す", func(t *testing.T) {
		profileID, actorID := mkTarget("active")

		if err := attemptInsertExport(t, tx, exportInsert{
			profileID: profileID, actorID: actorID, status: "queued",
		}); err != nil {
			t.Fatalf("最初のqueuedの挿入に失敗: %v", err)
		}

		// 同一プロフィールの2件目のactive (started) エクスポートは部分
		// ユニークインデックスに衝突する。
		err := attemptInsertExport(t, tx, exportInsert{
			profileID: profileID, actorID: actorID, status: "started", startedAt: ts,
		})
		if err == nil || !strings.Contains(err.Error(), "index_exports_on_profile_id_where_active") {
			t.Errorf("index_exports_on_profile_id_where_active違反を期待したが: %v", err)
		}

		// succeeded / failedの行は部分インデックスの対象外なので、activeな行と
		// 複数共存できる。
		for i := 0; i < 2; i++ {
			if err := attemptInsertExport(t, tx, exportInsert{
				profileID: profileID, actorID: actorID, status: "succeeded",
				objectKey: objectKey, startedAt: ts, finishedAt: ts,
			}); err != nil {
				t.Errorf("succeededの共存を期待したが%d件目で失敗: %v", i+1, err)
			}
		}
	})

	t.Run("検索と集計のインデックスを作成する", func(t *testing.T) {
		// これらは性能用インデックスで挿入では観測できないため、代わりに
		// カタログで存在を確認する。index_exports_on_profile_id_where_activeは上の
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
				t.Fatalf("%sの存在確認クエリに失敗: %v", name, err)
			}
			if !found {
				t.Errorf("インデックス%sが存在しない", name)
			}
		}
	})

	t.Run("完全なkeyset順序を持つ回復用インデックスを作成する", func(t *testing.T) {
		// 各回復cursorは時刻とidを行値として比較する。完全な並び順を
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
				t.Fatalf("%sの定義取得クエリに失敗: %v", name, err)
			}
			if !strings.Contains(indexDef, columns) {
				t.Errorf("インデックス%sのキーが不正: 実測値 = %q、%qを含むことを期待", name, indexDef, columns)
			}
		}
	})

	t.Run("エクスポートから参照されているactorの削除を阻止する", func(t *testing.T) {
		profileID, actorID := mkTarget("noaction")
		if err := attemptInsertExport(t, tx, exportInsert{
			profileID: profileID, actorID: actorID, status: "succeeded",
			objectKey: objectKey, startedAt: ts, finishedAt: ts,
		}); err != nil {
			t.Fatalf("succeededの挿入に失敗: %v", err)
		}

		if _, err := tx.Exec("SAVEPOINT sp_delete"); err != nil {
			t.Fatalf("SAVEPOINTの作成に失敗: %v", err)
		}
		_, err := tx.Exec("DELETE FROM actors WHERE id = $1", uuid.UUID(actorID))
		if _, rbErr := tx.Exec("ROLLBACK TO SAVEPOINT sp_delete"); rbErr != nil {
			t.Fatalf("ROLLBACK TO SAVEPOINTに失敗: %v", rbErr)
		}
		// ON DELETE NO ACTIONにより、エクスポートが参照している間はactorを
		// 削除できない。CASCADEなら代わりに削除されてしまう。
		if err == nil || !strings.Contains(err.Error(), "exports_actor_profile_fkey") {
			t.Errorf("exports_actor_profile_fkeyによる削除拒否を期待したが: %v", err)
		}
	})
}
