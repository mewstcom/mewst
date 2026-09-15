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

// TestExportCompletionNotificationRepository_FindAndMarkSentは配信1回分を
// 対象とする。senderはsucceededのエクスポートが残したsnapshotを読み、通知の退役は
// outbox行の削除と、exportのlegacyなcompletion_notified_at列へのmirrorの両方を
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
		t.Fatalf("FindByExportID()のエラー = %v", err)
	}
	if got == nil {
		t.Fatal("FindByExportID() = nil、通知を期待")
	}
	if got.ExportID != exportID {
		t.Errorf("got.ExportID = %v、期待値 = %v", got.ExportID, exportID)
	}
	if got.ActorID != actorID {
		t.Errorf("got.ActorID = %v、期待値 = %v", got.ActorID, actorID)
	}
	if got.RecipientEmail != recipientEmail {
		t.Errorf("got.RecipientEmail = %q、期待値 = %q", got.RecipientEmail, recipientEmail)
	}
	if got.Locale != locale {
		t.Errorf("got.Locale = %q、期待値 = %q", got.Locale, locale)
	}
	if !got.CreatedAt.Equal(createdAt) {
		t.Errorf("got.CreatedAt = %v、期待値 = %v", got.CreatedAt, createdAt)
	}

	if notifiedAt := exportCompletionNotifiedAt(t, tx, exportID); notifiedAt != nil {
		t.Errorf("送信前のcompletion_notified_at = %v、期待値 = nil", notifiedAt)
	}

	sent, err := repo.MarkSent(ctx, exportID)
	if err != nil {
		t.Fatalf("MarkSent()のエラー = %v", err)
	}
	if !sent {
		t.Fatal("MarkSent() = false、期待値 = true")
	}
	if notifiedAt := exportCompletionNotifiedAt(t, tx, exportID); notifiedAt == nil {
		t.Error("送信後のcompletion_notified_at = nil、時刻を期待")
	}

	// 同じ通知の2回目の配信は、退役させるものが残っていないと分かる。重複した
	// ジョブも、別の配信が既に行を消した実行も、どちらもこうなる。
	if sent, err := repo.MarkSent(ctx, exportID); err != nil || sent {
		t.Errorf("2回目のMarkSent() = (%v, %v)、期待値 = (false, nil)", sent, err)
	}
	if got, err := repo.FindByExportID(ctx, exportID); err != nil || got != nil {
		t.Errorf("MarkSent後のFindByExportID() = (%v, %v)、期待値 = (nil, nil)", got, err)
	}
}

// TestExportCompletionNotificationRepository_MarkSentWithoutExportは、保持
// cleanupにより通常となるケースを固定する。新しいエクスポートがこれを置き換えて
// その行は既に無く、通知だけが意図的に残っている状態である。配信は退役させる必要が
// あるため、export行が無いことで送信が「何も退役しなかった」となり、ジョブが永久に
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
		t.Fatalf("MarkSent()のエラー = %v", err)
	}
	if !sent {
		t.Error("MarkSent() = false、期待値 = true")
	}
	if got, err := repo.FindByExportID(ctx, exportID); err != nil || got != nil {
		t.Errorf("MarkSent後のFindByExportID() = (%v, %v)、期待値 = (nil, nil)", got, err)
	}
}

// TestExportCompletionNotificationRepository_MarkSentWithNonSucceededExportは
// mirrorが持つstatusの述語を固定する。exportsがcompletion_notified_atを
// 許すのはsucceededの行だけであり、mirrorが他の状態の行に届くと文全体が失敗して
// 配信済みのメールが退役できず、試行とリコンシリエーションを使い切るまで再送され
// 続ける。通知の退役はmirrorより優先される必要があり、述語はそれを文に行わせる。
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
		t.Fatalf("MarkSent()のエラー = %v", err)
	}
	if !sent {
		t.Error("MarkSent() = false、期待値 = true")
	}
	if notifiedAt := exportCompletionNotifiedAt(t, tx, exportID); notifiedAt != nil {
		t.Errorf("completion_notified_at = %v、期待値 = nil", notifiedAt)
	}
	if got, err := repo.FindByExportID(ctx, exportID); err != nil || got != nil {
		t.Errorf("MarkSent後のFindByExportID() = (%v, %v)、期待値 = (nil, nil)", got, err)
	}
}

// exportCompletionNotifiedAtはlegacyなmirror列を直接読む。この列をExport
// モデルに持たせていないのは意図的である。列は切り戻し互換のためだけのもので、本番の
// コードはこれを通知状態として読まない。
func exportCompletionNotifiedAt(t *testing.T, tx *sql.Tx, exportID model.ExportID) *time.Time {
	t.Helper()

	var notifiedAt sql.NullTime
	if err := tx.QueryRow(
		`SELECT completion_notified_at FROM exports WHERE id = $1`,
		uuid.UUID(exportID),
	).Scan(&notifiedAt); err != nil {
		t.Fatalf("completion_notified_atの取得に失敗: %v", err)
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

	// 厳密なしきい値により、境界上とそれより後の処理は除外する。
	add(threshold)
	add(threshold.Add(time.Hour))

	// 削除が始まったプロフィールの通知: 配信はマーカーで止まるため、この行を
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
		t.Fatalf("ListPending()のエラー = %v", err)
	}
	assertNotificationIDs(t, got, oldest, tied[0], tied[1])
	if next != nil {
		t.Errorf("ListPending()のnext = %v、期待値 = nil", next)
	}

	firstPage, cursor, err := repo.ListPending(ctx, threshold, nil, 2)
	if err != nil {
		t.Fatalf("1ページ目のListPending()のエラー = %v", err)
	}
	assertNotificationIDs(t, firstPage, oldest, tied[0])
	if cursor == nil {
		t.Fatal("ListPending()のカーソル = nil、カーソルを期待")
	}
	if !cursor.CreatedAt.Equal(firstPage[1].CreatedAt) || cursor.ExportID != firstPage[1].ExportID {
		t.Errorf("カーソル = %+v、期待値 = (%v, %v)", cursor, firstPage[1].CreatedAt, firstPage[1].ExportID)
	}

	secondPage, next, err := repo.ListPending(ctx, threshold, cursor, 2)
	if err != nil {
		t.Fatalf("2ページ目のListPending()のエラー = %v", err)
	}
	assertNotificationIDs(t, secondPage, tied[1])
	if next != nil {
		t.Errorf("2ページ目のListPending()のnext = %v、期待値 = nil", next)
	}
}

// TestExportCompletionNotificationRepository_DeleteByProfileIDは、プロフィール
// の削除が何を取り消すかを固定する。通知はexportではなく通知にsnapshotされた
// プロフィールから辿る。export行は先に削除され、通知は意図的にそれより長く残るため
// である。したがって、どの申請者が作成した通知かによって対象が狭まることはない。
func TestExportCompletionNotificationRepository_DeleteByProfileID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()
	repo := repository.NewExportCompletionNotificationRepository(testutil.QueriesWithTx(tx))

	owner := testutil.NewProfileOwner(t, tx)
	profileID, actorID := owner.ProfileID, owner.ActorID

	// プロフィールは複数のactorを持ちうるため、取り消しは最初に見つかった
	// 1つで止まってはならない。
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
		t.Fatalf("DeleteByProfileID()のエラー = %v", err)
	}
	if deleted != 2 {
		t.Errorf("DeleteByProfileID() = %d、期待値 = 2", deleted)
	}
	for _, exportID := range []model.ExportID{first, second} {
		if got, err := repo.FindByExportID(ctx, exportID); err != nil || got != nil {
			t.Errorf("削除後のFindByExportID(%v) = (%v, %v)、期待値 = (nil, nil)", exportID, got, err)
		}
	}
	if got, err := repo.FindByExportID(ctx, other); err != nil || got == nil {
		t.Errorf("他プロフィールのFindByExportID() = (%v, %v)、期待値 = (通知, nil)", got, err)
	}

	// 取り消すものが残っていないプロフィールは、失敗ではなく0を返す。ここまで
	// 進んだ削除の再実行がこれにあたる。
	deleted, err = repo.DeleteByProfileID(ctx, profileID)
	if err != nil {
		t.Fatalf("2回目のDeleteByProfileID()のエラー = %v", err)
	}
	if deleted != 0 {
		t.Errorf("2回目のDeleteByProfileID() = %d、期待値 = 0", deleted)
	}
}

func assertNotificationIDs(t *testing.T, notifications []*model.ExportCompletionNotification, want ...model.ExportID) {
	t.Helper()

	got := make([]model.ExportID, len(notifications))
	for i, notification := range notifications {
		got[i] = notification.ExportID
	}
	if !slices.Equal(got, want) {
		t.Errorf("通知のID = %v、期待値 = %v", got, want)
	}
}
