package seed

import (
	"context"
	"database/sql"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

// storedExportは、データベースから読み戻したエクスポートの行1件。
type storedExport struct {
	id           model.ExportID
	actorID      model.ActorID
	status       model.ExportStatus
	objectKey    *string
	attemptCount int32
	startedAt    *time.Time
	finishedAt   *time.Time
}

// storedExportPostは、エクスポートの投稿snapshotの行1件。
type storedExportPost struct {
	content     string
	publishedAt time.Time
}

// TestExportStatesは、実行が残すエクスポートが、画面を読むための状態を
// 覆っていること、1つのアカウントが同時に2件を持たされていないこと、そして
// エクスポートを持つ役割が、その画面を開ける役割であることを検証する。
func TestExportStates(t *testing.T) {
	t.Parallel()

	seenRole := make(map[seedRole]bool, len(exportStates))
	seenStatus := make(map[model.ExportStatus]bool, len(exportStates))

	for _, state := range exportStates {
		if !slices.Contains(allSeedRoles, state.role) {
			t.Errorf("exportStatesに名簿の持たない役割%sが含まれる", state.role)
		}

		// プロフィールが同時に持てるqueued / startedのエクスポートは1件
		// までであるため、2度名指しされた役割は、2つの状態を持つ実行ではなく、
		// 2件目の作成を拒まれる実行になる。
		if seenRole[state.role] {
			t.Errorf("役割%sがexportStatesに重複している", state.role)
		}
		seenRole[state.role] = true

		// succeededのエクスポートは、実行が書き込まないオブジェクト
		// ストレージ上のアーカイブのダウンロードを提示するため、その状態の行は、
		// ダウンロードが失敗する画面になる。
		if state.status == model.ExportStatusSucceeded {
			t.Errorf("役割%sへ%sのエクスポートが与えられている", state.role, state.status)
		}
		seenStatus[state.status] = true

		// roleNewcomerは何も書いていないため、そのエクスポートは中身の無い
		// アーカイブを表すことになる。roleDiscardedの削除済みプロフィールのポストは
		// どの画面にも無く、それをまとめたエクスポートも同様である。
		if state.role == roleNewcomer || state.role == roleDiscarded {
			t.Errorf("役割%sへエクスポートが与えられている", state.role)
		}
	}

	for _, status := range []model.ExportStatus{
		model.ExportStatusQueued,
		model.ExportStatusStarted,
		model.ExportStatusFailed,
	} {
		if !seenStatus[status] {
			t.Errorf("状態%sのエクスポートを持つ役割が1つも無い", status)
		}
	}
}

// TestCreateExportsは、エクスポートを持つそれぞれの役割が、示すためにいる
// 状態で残されること、その状態へ生成ジョブと同じ経路で到達すること、そしてそれが
// 表すアーカイブが、そのプロフィール自身のポストであることを検証する。
func TestCreateExports(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Microsecond)

	accounts, err := createAccounts(ctx, tx, newTestRoster(t), now)
	if err != nil {
		t.Fatalf("アカウントの作成に失敗: %v", err)
	}

	applicationID := testutil.NewOauthApplicationBuilder(t, tx).Build()
	if err := createPosts(ctx, tx, testAmounts, applicationID, accounts, now); err != nil {
		t.Fatalf("ポストの作成に失敗: %v", err)
	}
	if err := createLinks(ctx, tx, applicationID, accounts, now); err != nil {
		t.Fatalf("リンクの作成に失敗: %v", err)
	}

	if err := createExports(ctx, tx, accounts); err != nil {
		t.Fatalf("エクスポートの作成に失敗: %v", err)
	}

	for _, role := range allSeedRoles {
		account, err := accountForRole(accounts, role)
		if err != nil {
			t.Fatalf("役割%sのアカウントの取得に失敗: %v", role, err)
		}

		exports := readExports(t, ctx, tx, account.profile.ID)
		state, holds := exportStateForRole(role)

		if !holds {
			if len(exports) != 0 {
				t.Errorf("役割%sのエクスポートの件数 = %d、期待値 = 0", role, len(exports))
			}
			continue
		}

		if len(exports) != 1 {
			t.Fatalf("役割%sのエクスポートの件数 = %d、期待値 = 1", role, len(exports))
		}

		export := exports[0]
		if export.status != state.status {
			t.Errorf("役割%sのエクスポートの状態 = %s、期待値 = %s", role, export.status, state.status)
		}

		// 申請者はそのアカウント自身のactorである。複合外部キーは別の
		// プロフィールに属するactorを拒否するため、ここでの食い違いは、画面が持ち
		// うる行ではなく、シードがアカウントの誤った側を名指ししていることを意味する。
		if export.actorID != account.actor.ID {
			t.Errorf("役割%sのエクスポートの申請者 = %s、期待値 = %s",
				role, export.actorID, account.actor.ID)
		}

		assertExportStateFields(t, role, export)
		assertExportSnapshot(t, ctx, tx, role, export, account.profile.ID)
	}
}

// TestCreateExports_ProfileWithoutPostsは、snapshotが空になった
// エクスポートが、残されるのではなく報告されることを検証する。リコンシリエーションが
// それから作るのは、その行が表すアーカイブではなく空のzipであるため。
func TestCreateExports_ProfileWithoutPosts(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Microsecond)

	accounts, err := createAccounts(ctx, tx, newTestRoster(t), now)
	if err != nil {
		t.Fatalf("アカウントの作成に失敗: %v", err)
	}

	account, err := accountForRole(accounts, roleNewcomer)
	if err != nil {
		t.Fatalf("役割%sのアカウントの取得に失敗: %v", roleNewcomer, err)
	}

	writer := &exportWriter{tx: tx, exports: repository.NewExportRepository(query.New(tx))}

	err = writer.writeExport(ctx, account, model.ExportStatusQueued)
	if err == nil {
		t.Fatal("ポストを持たないプロフィールでエラーを返さなかった")
	}
	if want := "snapshotに含まれるポストが1件もありません"; !strings.Contains(err.Error(), want) {
		t.Errorf("エラー = %q、%qを含むことを期待", err, want)
	}
}

// TestExportTransition_UnsupportedStatusは、シードが辿る経路を持たない状態が、
// 黙って別の状態にされるのではなく報告されることを検証する。
func TestExportTransition_UnsupportedStatus(t *testing.T) {
	t.Parallel()

	writer := &exportWriter{}

	err := writer.transition(context.Background(), &model.Export{}, model.ExportStatusSucceeded)
	if err == nil {
		t.Fatal("シードが作れない状態でエラーを返さなかった")
	}
	if want := "はシードが作れません"; !strings.Contains(err.Error(), want) {
		t.Errorf("エラー = %q、%qを含むことを期待", err, want)
	}
}

// assertExportStateFieldsは、タイムスタンプ・オブジェクトキー・試行回数が、
// エクスポートの状態と整合していることを検証する。
//
// 試行回数は、エクスポートが生成ジョブと同じ経路でその状態へ到達したことを述べる
// ものである。シード独自の文でそこへ移された行は、状態を示しながら、試行が一度も
// 行われていないことを報告することになる。
func assertExportStateFields(t *testing.T, role seedRole, export storedExport) {
	t.Helper()

	if export.objectKey != nil {
		t.Errorf("役割%sのエクスポートのobject_key = %q、期待値 = なし", role, *export.objectKey)
	}

	switch export.status {
	case model.ExportStatusQueued:
		if export.attemptCount != 0 {
			t.Errorf("役割%sのqueuedのエクスポートの試行回数 = %d、期待値 = 0", role, export.attemptCount)
		}
		if export.startedAt != nil {
			t.Errorf("役割%sのqueuedのエクスポートにstarted_atがある", role)
		}
		if export.finishedAt != nil {
			t.Errorf("役割%sのqueuedのエクスポートにfinished_atがある", role)
		}
	case model.ExportStatusStarted:
		if export.attemptCount != 1 {
			t.Errorf("役割%sのstartedのエクスポートの試行回数 = %d、期待値 = 1", role, export.attemptCount)
		}
		if export.startedAt == nil {
			t.Errorf("役割%sのstartedのエクスポートにstarted_atがない", role)
		}
		if export.finishedAt != nil {
			t.Errorf("役割%sのstartedのエクスポートにfinished_atがある", role)
		}
	case model.ExportStatusFailed:
		if export.attemptCount != 1 {
			t.Errorf("役割%sのfailedのエクスポートの試行回数 = %d、期待値 = 1", role, export.attemptCount)
		}
		if export.startedAt == nil {
			t.Errorf("役割%sのfailedのエクスポートにstarted_atがない", role)
		}
		if export.finishedAt == nil {
			t.Errorf("役割%sのfailedのエクスポートにfinished_atがない", role)
		}
	default:
		t.Errorf("役割%sのエクスポートが状態%sで残されている", role, export.status)
	}
}

// assertExportSnapshotは、エクスポートの投稿snapshotを、そのプロフィールが
// 持つポストと突き合わせる。
//
// snapshotを期待するのは、エクスポートがまだそれを元に生成されうる間だけである。
// 終端状態への到達は同じ文でsnapshotを破棄するため、それを持つfailedの
// エクスポートは、アプリケーションが決して残さない行になる。
func assertExportSnapshot(
	t *testing.T,
	ctx context.Context,
	tx *sql.Tx,
	role seedRole,
	export storedExport,
	profileID model.ProfileID,
) {
	t.Helper()

	snapshot := readExportPosts(t, ctx, tx, export.id)

	if export.status == model.ExportStatusFailed {
		if len(snapshot) != 0 {
			t.Errorf("役割%sのfailedのエクスポートのsnapshotの件数 = %d、期待値 = 0", role, len(snapshot))
		}
		return
	}

	kept := readKeptPosts(t, ctx, tx, profileID)
	if len(kept) == 0 {
		t.Fatalf("役割%sのプロフィールがsnapshotと突き合わせるポストを持たない", role)
	}

	if len(snapshot) != len(kept) {
		t.Errorf("役割%sのエクスポートのsnapshotの件数 = %d、期待値 = %d", role, len(snapshot), len(kept))
	}

	for id, want := range kept {
		got, ok := snapshot[id]
		if !ok {
			t.Errorf("役割%sのポスト%sがエクスポートのsnapshotにない", role, id)
			continue
		}
		if got.content != want.content {
			t.Errorf("役割%sのポスト%sのsnapshotの本文 = %q、期待値 = %q", role, id, got.content, want.content)
		}
		if !got.publishedAt.Equal(want.publishedAt) {
			t.Errorf("役割%sのポスト%sのsnapshotの公開日時 = %v、期待値 = %v",
				role, id, got.publishedAt, want.publishedAt)
		}
	}
}

// readExportsは、あるプロフィールのエクスポートを古い順に返す。
func readExports(t *testing.T, ctx context.Context, tx *sql.Tx, profileID model.ProfileID) []storedExport {
	t.Helper()

	rows, err := tx.QueryContext(ctx, `
		SELECT id, actor_id, status, object_key, attempt_count, started_at, finished_at
		FROM exports
		WHERE profile_id = $1
		ORDER BY created_at ASC, id ASC
	`, uuid.UUID(profileID))
	if err != nil {
		t.Fatalf("エクスポートの取得に失敗: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var exports []storedExport
	for rows.Next() {
		var (
			id         uuid.UUID
			actorID    uuid.UUID
			status     string
			objectKey  sql.NullString
			attempts   int32
			startedAt  sql.NullTime
			finishedAt sql.NullTime
		)
		if err := rows.Scan(&id, &actorID, &status, &objectKey, &attempts, &startedAt, &finishedAt); err != nil {
			t.Fatalf("エクスポートの読み取りに失敗: %v", err)
		}

		export := storedExport{
			id:           model.ExportID(id),
			actorID:      model.ActorID(actorID),
			status:       model.ExportStatus(status),
			attemptCount: attempts,
		}
		if objectKey.Valid {
			export.objectKey = &objectKey.String
		}
		if startedAt.Valid {
			export.startedAt = &startedAt.Time
		}
		if finishedAt.Valid {
			export.finishedAt = &finishedAt.Time
		}

		exports = append(exports, export)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("エクスポートの走査に失敗: %v", err)
	}

	return exports
}

// readExportPostsは、エクスポートの投稿snapshotを、各行の複製元となった
// ポストで引ける形で返す。
func readExportPosts(
	t *testing.T,
	ctx context.Context,
	tx *sql.Tx,
	exportID model.ExportID,
) map[model.PostID]storedExportPost {
	t.Helper()

	rows, err := tx.QueryContext(ctx, `
		SELECT post_id, content, published_at
		FROM export_posts
		WHERE export_id = $1
	`, uuid.UUID(exportID))
	if err != nil {
		t.Fatalf("エクスポートのsnapshotの取得に失敗: %v", err)
	}
	defer func() { _ = rows.Close() }()

	posts := make(map[model.PostID]storedExportPost)
	for rows.Next() {
		var (
			postID uuid.UUID
			post   storedExportPost
		)
		if err := rows.Scan(&postID, &post.content, &post.publishedAt); err != nil {
			t.Fatalf("エクスポートのsnapshotの読み取りに失敗: %v", err)
		}
		posts[model.PostID(postID)] = post
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("エクスポートのsnapshotの走査に失敗: %v", err)
	}

	return posts
}

// readKeptPostsは、あるプロフィールの削除されていないポストを返す。
// エクスポートの生成元となるのがそれである。
func readKeptPosts(
	t *testing.T,
	ctx context.Context,
	tx *sql.Tx,
	profileID model.ProfileID,
) map[model.PostID]storedExportPost {
	t.Helper()

	rows, err := tx.QueryContext(ctx, `
		SELECT id, content, published_at
		FROM posts
		WHERE profile_id = $1 AND discarded_at IS NULL
	`, uuid.UUID(profileID))
	if err != nil {
		t.Fatalf("ポストの取得に失敗: %v", err)
	}
	defer func() { _ = rows.Close() }()

	posts := make(map[model.PostID]storedExportPost)
	for rows.Next() {
		var (
			postID uuid.UUID
			post   storedExportPost
		)
		if err := rows.Scan(&postID, &post.content, &post.publishedAt); err != nil {
			t.Fatalf("ポストの読み取りに失敗: %v", err)
		}
		posts[model.PostID(postID)] = post
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("ポストの走査に失敗: %v", err)
	}

	return posts
}
