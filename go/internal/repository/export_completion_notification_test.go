package repository_test

import (
	"context"
	"database/sql"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

// TestExportCompletionNotificationRepository_FindAndMarkSent covers one
// delivery: the sender reads the snapshot the succeeded export left, and
// retiring the notification both deletes the outbox row and mirrors the send
// into the export's legacy completion_notified_at column.
//
// [Ja] TestExportCompletionNotificationRepository_FindAndMarkSent は配信 1 回分を
// 対象とする。sender は succeeded のエクスポートが残した snapshot を読み、通知の退役は
// outbox 行の削除と、export の legacy な completion_notified_at 列への mirror の両方を
// 行う。
func TestExportCompletionNotificationRepository_FindAndMarkSent(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()
	owner := testutil.NewProfileOwner(t, tx)
	profileID, actorID := owner.ProfileID, owner.ActorID
	exportID := testutil.NewExportBuilder(t, tx).
		WithProfileID(profileID).
		WithActorID(actorID).
		WithStatus(model.ExportStatusSucceeded).
		Build()
	repo := repository.NewExportCompletionNotificationRepository(testutil.QueriesWithTx(tx))

	const (
		recipientEmail = "snapshot@example.com"
		locale         = "en"
	)
	createdAt := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	testutil.NewExportCompletionNotificationBuilder(t, tx).
		WithExportID(exportID).
		WithActorID(actorID).
		WithRecipientEmail(recipientEmail).
		WithLocale(locale).
		WithCreatedAt(createdAt).
		Build()

	got, err := repo.FindByExportID(ctx, exportID)
	if err != nil {
		t.Fatalf("FindByExportID() error = %v", err)
	}
	if got == nil {
		t.Fatal("FindByExportID() = nil, want a notification")
	}
	if got.ExportID != exportID {
		t.Errorf("got.ExportID = %v, want %v", got.ExportID, exportID)
	}
	if got.ActorID != actorID {
		t.Errorf("got.ActorID = %v, want %v", got.ActorID, actorID)
	}
	if got.RecipientEmail != recipientEmail {
		t.Errorf("got.RecipientEmail = %q, want %q", got.RecipientEmail, recipientEmail)
	}
	if got.Locale != locale {
		t.Errorf("got.Locale = %q, want %q", got.Locale, locale)
	}
	if !got.CreatedAt.Equal(createdAt) {
		t.Errorf("got.CreatedAt = %v, want %v", got.CreatedAt, createdAt)
	}

	if notifiedAt := exportCompletionNotifiedAt(t, tx, exportID); notifiedAt != nil {
		t.Errorf("送信前の completion_notified_at = %v, want nil", notifiedAt)
	}

	sent, err := repo.MarkSent(ctx, exportID)
	if err != nil {
		t.Fatalf("MarkSent() error = %v", err)
	}
	if !sent {
		t.Fatal("MarkSent() = false, want true")
	}
	if notifiedAt := exportCompletionNotifiedAt(t, tx, exportID); notifiedAt == nil {
		t.Error("送信後の completion_notified_at = nil, want a timestamp")
	}

	// A second delivery of the same notification finds nothing left to retire,
	// which is how a duplicate job and a run whose row another delivery already
	// removed both end.
	//
	// [Ja] 同じ通知の 2 回目の配信は、退役させるものが残っていないと分かる。重複した
	// ジョブも、別の配信が既に行を消した実行も、どちらもこうなる。
	if sent, err := repo.MarkSent(ctx, exportID); err != nil || sent {
		t.Errorf("second MarkSent() = (%v, %v), want (false, nil)", sent, err)
	}
	if got, err := repo.FindByExportID(ctx, exportID); err != nil || got != nil {
		t.Errorf("FindByExportID() after MarkSent = (%v, %v), want (nil, nil)", got, err)
	}
}

// TestExportCompletionNotificationRepository_MarkSentWithoutExport pins the
// case retention cleanup makes the normal one: a newer export replaced this one
// and its row is already gone, while the notification deliberately outlived it.
// The delivery still has to be retired, so the missing export must not turn the
// send into "nothing was retired" and leave the job re-sending forever.
//
// [Ja] TestExportCompletionNotificationRepository_MarkSentWithoutExport は、保持
// cleanup により通常となるケースを固定する。新しいエクスポートがこれを置き換えて
// その行は既に無く、通知だけが意図的に残っている状態である。配信は退役させる必要が
// あるため、export 行が無いことで送信が「何も退役しなかった」となり、ジョブが永久に
// 再送し続ける状態にしてはならない。
func TestExportCompletionNotificationRepository_MarkSentWithoutExport(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()
	owner := testutil.NewProfileOwner(t, tx)
	actorID := owner.ActorID
	exportID := model.ExportID(uuid.New())
	repo := repository.NewExportCompletionNotificationRepository(testutil.QueriesWithTx(tx))

	testutil.NewExportCompletionNotificationBuilder(t, tx).
		WithExportID(exportID).
		WithActorID(actorID).
		Build()

	sent, err := repo.MarkSent(ctx, exportID)
	if err != nil {
		t.Fatalf("MarkSent() error = %v", err)
	}
	if !sent {
		t.Error("MarkSent() = false, want true")
	}
	if got, err := repo.FindByExportID(ctx, exportID); err != nil || got != nil {
		t.Errorf("FindByExportID() after MarkSent = (%v, %v), want (nil, nil)", got, err)
	}
}

// TestExportCompletionNotificationRepository_MarkSentWithNonSucceededExport
// pins the status predicate the mirror carries. exports only accepts
// completion_notified_at on a succeeded row, so a mirror that reached a row in
// any other state would fail the whole statement and leave a delivered email
// unretired, re-sending it until the attempts and then the reconciliation run
// out. Retiring the notification has to win over the mirror, which is what the
// predicate makes the statement do.
//
// [Ja] TestExportCompletionNotificationRepository_MarkSentWithNonSucceededExport は
// mirror が持つ status の述語を固定する。exports が completion_notified_at を
// 許すのは succeeded の行だけであり、mirror が他の状態の行に届くと文全体が失敗して
// 配信済みのメールが退役できず、試行とリコンシリエーションを使い切るまで再送され
// 続ける。通知の退役は mirror より優先される必要があり、述語はそれを文に行わせる。
func TestExportCompletionNotificationRepository_MarkSentWithNonSucceededExport(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()
	owner := testutil.NewProfileOwner(t, tx)
	profileID, actorID := owner.ProfileID, owner.ActorID
	exportID := testutil.NewExportBuilder(t, tx).
		WithProfileID(profileID).
		WithActorID(actorID).
		WithStatus(model.ExportStatusQueued).
		Build()
	repo := repository.NewExportCompletionNotificationRepository(testutil.QueriesWithTx(tx))

	testutil.NewExportCompletionNotificationBuilder(t, tx).
		WithExportID(exportID).
		WithActorID(actorID).
		Build()

	sent, err := repo.MarkSent(ctx, exportID)
	if err != nil {
		t.Fatalf("MarkSent() error = %v", err)
	}
	if !sent {
		t.Error("MarkSent() = false, want true")
	}
	if notifiedAt := exportCompletionNotifiedAt(t, tx, exportID); notifiedAt != nil {
		t.Errorf("completion_notified_at = %v, want nil", notifiedAt)
	}
	if got, err := repo.FindByExportID(ctx, exportID); err != nil || got != nil {
		t.Errorf("FindByExportID() after MarkSent = (%v, %v), want (nil, nil)", got, err)
	}
}

// exportCompletionNotifiedAt reads the legacy mirror column directly. It is
// deliberately not on the Export model: the column is downgrade compatibility
// only, and no production code reads it as the notification state.
//
// [Ja] exportCompletionNotifiedAt は legacy な mirror 列を直接読む。この列を Export
// モデルに持たせていないのは意図的である。列は切り戻し互換のためだけのもので、本番の
// コードはこれを通知状態として読まない。
func exportCompletionNotifiedAt(t *testing.T, tx *sql.Tx, exportID model.ExportID) *time.Time {
	t.Helper()

	var notifiedAt sql.NullTime
	if err := tx.QueryRow(
		`SELECT completion_notified_at FROM exports WHERE id = $1`,
		uuid.UUID(exportID),
	).Scan(&notifiedAt); err != nil {
		t.Fatalf("completion_notified_at の取得に失敗: %v", err)
	}
	if !notifiedAt.Valid {
		return nil
	}
	return &notifiedAt.Time
}

func TestExportCompletionNotificationRepository_ListPending(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTxRepeatableRead(t)
	ctx := context.Background()
	owner := testutil.NewProfileOwner(t, tx)
	actorID := owner.ActorID
	repo := repository.NewExportCompletionNotificationRepository(testutil.QueriesWithTx(tx))
	threshold := time.Date(1900, 1, 3, 0, 0, 0, 0, time.UTC)

	add := func(createdAt time.Time) model.ExportID {
		t.Helper()
		exportID := model.ExportID(uuid.New())
		testutil.NewExportCompletionNotificationBuilder(t, tx).
			WithExportID(exportID).
			WithActorID(actorID).
			WithCreatedAt(createdAt).
			Build()
		return exportID
	}

	oldest := add(time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC))
	tied := []model.ExportID{
		add(time.Date(1900, 1, 2, 0, 0, 0, 0, time.UTC)),
		add(time.Date(1900, 1, 2, 0, 0, 0, 0, time.UTC)),
	}
	sortExportIDs(tied)

	// The strict threshold excludes both the boundary and later work.
	//
	// [Ja] 厳密なしきい値により、境界上とそれより後の処理は除外する。
	add(threshold)
	add(threshold.Add(time.Hour))

	// A notification of a profile whose deletion has started: delivery stops at
	// the marker, so returning this row would only produce a job that sends
	// nothing. Cancelling it is profile deletion's work.
	//
	// [Ja] 削除が始まったプロフィールの通知: 配信はマーカーで止まるため、この行を
	// 返しても、何も送らないジョブが生まれるだけである。取り消すのは親削除の仕事。
	deletingOwner := testutil.NewProfileOwner(t, tx)
	deletingProfile, deletingActor := deletingOwner.ProfileID, deletingOwner.ActorID
	testutil.NewExportCompletionNotificationBuilder(t, tx).
		WithExportID(model.ExportID(uuid.New())).
		WithActorID(deletingActor).
		WithCreatedAt(time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)).
		Build()
	if _, err := tx.Exec(
		"UPDATE profiles SET export_deletion_started_at = NOW() WHERE id = $1",
		uuid.UUID(deletingProfile),
	); err != nil {
		t.Fatalf("プロフィール削除開始の記録に失敗: %v", err)
	}

	got, next, err := repo.ListPending(ctx, threshold, nil, unboundedTestPageSize)
	if err != nil {
		t.Fatalf("ListPending() error = %v", err)
	}
	assertNotificationIDs(t, got, oldest, tied[0], tied[1])
	if next != nil {
		t.Errorf("ListPending() next = %v, want nil", next)
	}

	firstPage, cursor, err := repo.ListPending(ctx, threshold, nil, 2)
	if err != nil {
		t.Fatalf("ListPending() first page error = %v", err)
	}
	assertNotificationIDs(t, firstPage, oldest, tied[0])
	if cursor == nil {
		t.Fatal("ListPending() cursor = nil, want a cursor")
	}
	if !cursor.CreatedAt.Equal(firstPage[1].CreatedAt) || cursor.ExportID != firstPage[1].ExportID {
		t.Errorf("cursor = %+v, want (%v, %v)", cursor, firstPage[1].CreatedAt, firstPage[1].ExportID)
	}

	secondPage, next, err := repo.ListPending(ctx, threshold, cursor, 2)
	if err != nil {
		t.Fatalf("ListPending() second page error = %v", err)
	}
	assertNotificationIDs(t, secondPage, tied[1])
	if next != nil {
		t.Errorf("ListPending() second next = %v, want nil", next)
	}
}

// TestExportCompletionNotificationRepository_DeleteByProfileID pins what a
// profile removal cancels. The notifications are reached through the profile
// snapshotted on them rather than through the export, because the export row is
// deleted first and the notification deliberately outlives it, so the requester
// that created a notification does not narrow what is cancelled.
//
// [Ja] TestExportCompletionNotificationRepository_DeleteByProfileID は、プロフィール
// の削除が何を取り消すかを固定する。通知は export ではなく通知に snapshot された
// プロフィールから辿る。export 行は先に削除され、通知は意図的にそれより長く残るため
// である。したがって、どの申請者が作成した通知かによって対象が狭まることはない。
func TestExportCompletionNotificationRepository_DeleteByProfileID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()
	repo := repository.NewExportCompletionNotificationRepository(testutil.QueriesWithTx(tx))

	owner := testutil.NewProfileOwner(t, tx)
	profileID, actorID := owner.ProfileID, owner.ActorID

	// A profile can hold more than one actor, so the cancellation cannot stop at
	// the one that happens to be found first.
	//
	// [Ja] プロフィールは複数の actor を持ちうるため、取り消しは最初に見つかった
	// 1 つで止まってはならない。
	secondActorID := testutil.NewActorBuilder(t, tx).
		WithUserID(testutil.NewUserBuilder(t, tx).Build()).
		WithProfileID(profileID).
		Build()
	otherOwner := testutil.NewProfileOwner(t, tx)
	otherActorID := otherOwner.ActorID

	add := func(requesterID model.ActorID) model.ExportID {
		t.Helper()
		exportID := model.ExportID(uuid.New())
		testutil.NewExportCompletionNotificationBuilder(t, tx).
			WithExportID(exportID).
			WithActorID(requesterID).
			Build()
		return exportID
	}

	first := add(actorID)
	second := add(secondActorID)
	other := add(otherActorID)

	deleted, err := repo.DeleteByProfileID(ctx, profileID)
	if err != nil {
		t.Fatalf("DeleteByProfileID() error = %v", err)
	}
	if deleted != 2 {
		t.Errorf("DeleteByProfileID() = %d, want 2", deleted)
	}
	for _, exportID := range []model.ExportID{first, second} {
		if got, err := repo.FindByExportID(ctx, exportID); err != nil || got != nil {
			t.Errorf("FindByExportID(%v) after delete = (%v, %v), want (nil, nil)", exportID, got, err)
		}
	}
	if got, err := repo.FindByExportID(ctx, other); err != nil || got == nil {
		t.Errorf("他プロフィールの FindByExportID() = (%v, %v), want (notification, nil)", got, err)
	}

	// A profile with nothing left to cancel reports zero rather than failing,
	// which is what a rerun of a deletion that already got this far does.
	//
	// [Ja] 取り消すものが残っていないプロフィールは、失敗ではなく 0 を返す。ここまで
	// 進んだ削除の再実行がこれにあたる。
	deleted, err = repo.DeleteByProfileID(ctx, profileID)
	if err != nil {
		t.Fatalf("2 回目の DeleteByProfileID() error = %v", err)
	}
	if deleted != 0 {
		t.Errorf("2 回目の DeleteByProfileID() = %d, want 0", deleted)
	}
}

func assertNotificationIDs(t *testing.T, notifications []*model.ExportCompletionNotification, want ...model.ExportID) {
	t.Helper()

	got := make([]model.ExportID, len(notifications))
	for i, notification := range notifications {
		got[i] = notification.ExportID
	}
	if !slices.Equal(got, want) {
		t.Errorf("notification IDs = %v, want %v", got, want)
	}
}
