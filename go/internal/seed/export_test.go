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

// storedExport is one export row read back out of the database.
//
// [Ja] storedExport は、データベースから読み戻したエクスポートの行 1 件。
type storedExport struct {
	id           model.ExportID
	actorID      model.ActorID
	status       model.ExportStatus
	objectKey    *string
	attemptCount int32
	startedAt    *time.Time
	finishedAt   *time.Time
}

// storedExportPost is one row of an export's post snapshot.
//
// [Ja] storedExportPost は、エクスポートの投稿 snapshot の行 1 件。
type storedExportPost struct {
	content     string
	publishedAt time.Time
}

// TestExportStates verifies that the exports a run leaves behind cover the
// statuses the screen is read in, that no account is asked to hold two of them
// at once, and that the roles holding one are roles whose export a screen can
// be opened on.
//
// [Ja] TestExportStates は、実行が残すエクスポートが、画面を読むための状態を
// 覆っていること、1 つのアカウントが同時に 2 件を持たされていないこと、そして
// エクスポートを持つ役割が、その画面を開ける役割であることを検証する。
func TestExportStates(t *testing.T) {
	t.Parallel()

	seenRole := make(map[seedRole]bool, len(exportStates))
	seenStatus := make(map[model.ExportStatus]bool, len(exportStates))

	for _, state := range exportStates {
		if !slices.Contains(allSeedRoles, state.role) {
			t.Errorf("exportStates に名簿の持たない役割 %s が含まれる", state.role)
		}

		// A profile is allowed one queued or started export at a time, so a
		// role named twice is a run whose second create is refused rather than
		// one that holds two states.
		//
		// [Ja] プロフィールが同時に持てる queued / started のエクスポートは 1 件
		// までであるため、2 度名指しされた役割は、2 つの状態を持つ実行ではなく、
		// 2 件目の作成を拒まれる実行になる。
		if seenRole[state.role] {
			t.Errorf("役割 %s が exportStates に重複している", state.role)
		}
		seenRole[state.role] = true

		// A succeeded export offers a download of an archive in the object
		// storage, which a run does not write, so a row in that status is a
		// screen whose download fails.
		//
		// [Ja] succeeded のエクスポートは、実行が書き込まないオブジェクト
		// ストレージ上のアーカイブのダウンロードを提示するため、その状態の行は、
		// ダウンロードが失敗する画面になる。
		if state.status == model.ExportStatusSucceeded {
			t.Errorf("役割 %s へ %s のエクスポートが与えられている", state.role, state.status)
		}
		seenStatus[state.status] = true

		// roleNewcomer has written nothing, so its export would stand for an
		// archive of nothing; the posts of roleDiscarded's deleted profile are
		// on no screen, and neither is an export of them.
		//
		// [Ja] roleNewcomer は何も書いていないため、そのエクスポートは中身の無い
		// アーカイブを表すことになる。roleDiscarded の削除済みプロフィールのポストは
		// どの画面にも無く、それをまとめたエクスポートも同様である。
		if state.role == roleNewcomer || state.role == roleDiscarded {
			t.Errorf("役割 %s へエクスポートが与えられている", state.role)
		}
	}

	for _, status := range []model.ExportStatus{
		model.ExportStatusQueued,
		model.ExportStatusStarted,
		model.ExportStatusFailed,
	} {
		if !seenStatus[status] {
			t.Errorf("状態 %s のエクスポートを持つ役割が 1 つも無い", status)
		}
	}
}

// TestCreateExports verifies that each role holding an export is left in the
// status it is there to show, that it reached that status the way the
// generation job does, and that the archive it stands for is the profile's own
// posts.
//
// [Ja] TestCreateExports は、エクスポートを持つそれぞれの役割が、示すためにいる
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
			t.Fatalf("役割 %s のアカウントの取得に失敗: %v", role, err)
		}

		exports := readExports(t, ctx, tx, account.profile.ID)
		state, holds := exportStateForRole(role)

		if !holds {
			if len(exports) != 0 {
				t.Errorf("役割 %s のエクスポートの件数 = %d, want 0", role, len(exports))
			}
			continue
		}

		if len(exports) != 1 {
			t.Fatalf("役割 %s のエクスポートの件数 = %d, want 1", role, len(exports))
		}

		export := exports[0]
		if export.status != state.status {
			t.Errorf("役割 %s のエクスポートの状態 = %s, want %s", role, export.status, state.status)
		}

		// The requester is the account's own actor. The composite foreign key
		// refuses an actor belonging to another profile, so a mismatch here is
		// the seed naming the wrong end of an account rather than a row a
		// screen could hold.
		//
		// [Ja] 申請者はそのアカウント自身の actor である。複合外部キーは別の
		// プロフィールに属する actor を拒否するため、ここでの食い違いは、画面が持ち
		// うる行ではなく、シードがアカウントの誤った側を名指ししていることを意味する。
		if export.actorID != account.actor.ID {
			t.Errorf("役割 %s のエクスポートの申請者 = %s, want %s",
				role, export.actorID, account.actor.ID)
		}

		assertExportStateFields(t, role, export)
		assertExportSnapshot(t, ctx, tx, role, export, account.profile.ID)
	}
}

// TestCreateExports_ProfileWithoutPosts verifies that an export whose snapshot
// came out empty is reported rather than left behind, because what
// reconciliation would make of it is an empty zip rather than the archive the
// row stands for.
//
// [Ja] TestCreateExports_ProfileWithoutPosts は、snapshot が空になった
// エクスポートが、残されるのではなく報告されることを検証する。リコンシリエーションが
// それから作るのは、その行が表すアーカイブではなく空の zip であるため。
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
		t.Fatalf("役割 %s のアカウントの取得に失敗: %v", roleNewcomer, err)
	}

	writer := &exportWriter{tx: tx, exports: repository.NewExportRepository(query.New(tx))}

	err = writer.writeExport(ctx, account, model.ExportStatusQueued)
	if err == nil {
		t.Fatal("ポストを持たないプロフィールでエラーを返さなかった")
	}
	if want := "snapshot に含まれるポストが 1 件もありません"; !strings.Contains(err.Error(), want) {
		t.Errorf("エラー = %q, want %q を含む", err, want)
	}
}

// TestExportTransition_UnsupportedStatus verifies that a status the seed has
// no path to is reported rather than quietly turned into another one.
//
// [Ja] TestExportTransition_UnsupportedStatus は、シードが辿る経路を持たない状態が、
// 黙って別の状態にされるのではなく報告されることを検証する。
func TestExportTransition_UnsupportedStatus(t *testing.T) {
	t.Parallel()

	writer := &exportWriter{}

	err := writer.transition(context.Background(), &model.Export{}, model.ExportStatusSucceeded)
	if err == nil {
		t.Fatal("シードが作れない状態でエラーを返さなかった")
	}
	if want := "はシードが作れません"; !strings.Contains(err.Error(), want) {
		t.Errorf("エラー = %q, want %q を含む", err, want)
	}
}

// assertExportStateFields verifies that the timestamps, the object key and the
// attempt count agree with the status the export is in.
//
// The attempt count is what says the export reached its status the way the
// generation job reaches it. A row moved there by a statement of the seed's
// own would show the status while reporting that nothing was ever attempted.
//
// [Ja] assertExportStateFields は、タイムスタンプ・オブジェクトキー・試行回数が、
// エクスポートの状態と整合していることを検証する。
//
// 試行回数は、エクスポートが生成ジョブと同じ経路でその状態へ到達したことを述べる
// ものである。シード独自の文でそこへ移された行は、状態を示しながら、試行が一度も
// 行われていないことを報告することになる。
func assertExportStateFields(t *testing.T, role seedRole, export storedExport) {
	t.Helper()

	if export.objectKey != nil {
		t.Errorf("役割 %s のエクスポートの object_key = %q, want なし", role, *export.objectKey)
	}

	switch export.status {
	case model.ExportStatusQueued:
		if export.attemptCount != 0 {
			t.Errorf("役割 %s の queued のエクスポートの試行回数 = %d, want 0", role, export.attemptCount)
		}
		if export.startedAt != nil {
			t.Errorf("役割 %s の queued のエクスポートに started_at がある", role)
		}
		if export.finishedAt != nil {
			t.Errorf("役割 %s の queued のエクスポートに finished_at がある", role)
		}
	case model.ExportStatusStarted:
		if export.attemptCount != 1 {
			t.Errorf("役割 %s の started のエクスポートの試行回数 = %d, want 1", role, export.attemptCount)
		}
		if export.startedAt == nil {
			t.Errorf("役割 %s の started のエクスポートに started_at がない", role)
		}
		if export.finishedAt != nil {
			t.Errorf("役割 %s の started のエクスポートに finished_at がある", role)
		}
	case model.ExportStatusFailed:
		if export.attemptCount != 1 {
			t.Errorf("役割 %s の failed のエクスポートの試行回数 = %d, want 1", role, export.attemptCount)
		}
		if export.startedAt == nil {
			t.Errorf("役割 %s の failed のエクスポートに started_at がない", role)
		}
		if export.finishedAt == nil {
			t.Errorf("役割 %s の failed のエクスポートに finished_at がない", role)
		}
	default:
		t.Errorf("役割 %s のエクスポートが状態 %s で残されている", role, export.status)
	}
}

// assertExportSnapshot compares an export's post snapshot with the posts its
// profile holds.
//
// The snapshot is only expected while the export can still be generated from
// it. Reaching a terminal status discards it in the same statement, so a
// failed export holding one would be a row the application never leaves
// behind.
//
// [Ja] assertExportSnapshot は、エクスポートの投稿 snapshot を、そのプロフィールが
// 持つポストと突き合わせる。
//
// snapshot を期待するのは、エクスポートがまだそれを元に生成されうる間だけである。
// 終端状態への到達は同じ文で snapshot を破棄するため、それを持つ failed の
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
			t.Errorf("役割 %s の failed のエクスポートの snapshot の件数 = %d, want 0", role, len(snapshot))
		}
		return
	}

	kept := readKeptPosts(t, ctx, tx, profileID)
	if len(kept) == 0 {
		t.Fatalf("役割 %s のプロフィールが snapshot と突き合わせるポストを持たない", role)
	}

	if len(snapshot) != len(kept) {
		t.Errorf("役割 %s のエクスポートの snapshot の件数 = %d, want %d", role, len(snapshot), len(kept))
	}

	for id, want := range kept {
		got, ok := snapshot[id]
		if !ok {
			t.Errorf("役割 %s のポスト %s がエクスポートの snapshot にない", role, id)
			continue
		}
		if got.content != want.content {
			t.Errorf("役割 %s のポスト %s の snapshot の本文 = %q, want %q", role, id, got.content, want.content)
		}
		if !got.publishedAt.Equal(want.publishedAt) {
			t.Errorf("役割 %s のポスト %s の snapshot の公開日時 = %v, want %v",
				role, id, got.publishedAt, want.publishedAt)
		}
	}
}

// readExports returns the exports of one profile, oldest first.
//
// [Ja] readExports は、あるプロフィールのエクスポートを古い順に返す。
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

// readExportPosts returns an export's post snapshot, by the post each row was
// copied from.
//
// [Ja] readExportPosts は、エクスポートの投稿 snapshot を、各行の複製元となった
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
		t.Fatalf("エクスポートの snapshot の取得に失敗: %v", err)
	}
	defer func() { _ = rows.Close() }()

	posts := make(map[model.PostID]storedExportPost)
	for rows.Next() {
		var (
			postID uuid.UUID
			post   storedExportPost
		)
		if err := rows.Scan(&postID, &post.content, &post.publishedAt); err != nil {
			t.Fatalf("エクスポートの snapshot の読み取りに失敗: %v", err)
		}
		posts[model.PostID(postID)] = post
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("エクスポートの snapshot の走査に失敗: %v", err)
	}

	return posts
}

// readKeptPosts returns the posts of a profile that have not been deleted,
// which are the ones an export is built from.
//
// [Ja] readKeptPosts は、あるプロフィールの削除されていないポストを返す。
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
