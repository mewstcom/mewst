package usecase_test

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// jobRecordはリコンシリエーションが試みた投入1件。ジョブが持つ識別子と、
// 同じ作業依頼のジョブが未完了のときにRiverが返す一意性によるskipだったかを持つ。
type jobRecord struct {
	kind    string
	id      string
	skipped bool
}

// exportJobInserterはエクスポートの回復系ジョブにおけるジョブキューの代役。
// 試みられた投入をすべて記録し、実行が走査しただけの候補と引き受けた候補をテストが
// 区別できるようにする。また、一意性によるskipやエラーを返すことで、混み合ったキューや
// 壊れたキューでしか生じない状態を作れる。
type exportJobInserter struct {
	t *testing.T

	// skipIDsは一意性によるskipを返す識別子。その作業依頼のジョブがすでに
	// 待機中または実行中であることを表す。
	skipIDs map[string]bool

	// failKindsはキューが受け付けないジョブ種別。その種類の処理についてキューへ
	// 到達できない状態を表す。
	failKinds map[string]bool

	mu      sync.Mutex
	records []jobRecord
}

func newExportJobInserter(t *testing.T) *exportJobInserter {
	t.Helper()
	return &exportJobInserter{
		t:         t,
		skipIDs:   map[string]bool{},
		failKinds: map[string]bool{},
	}
}

func (i *exportJobInserter) Insert(_ context.Context, args river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	i.t.Helper()

	record := jobRecord{kind: args.Kind()}
	switch a := args.(type) {
	case dispatcher.GenerateExportArgs:
		record.id = a.ExportID
	case dispatcher.SendExportCompletedEmailArgs:
		record.id = a.ExportID
	case dispatcher.CleanupOldExportsArgs:
		record.id = a.ProfileID
	case dispatcher.CleanupOrphanExportObjectsArgs:
		// 掃除は継続ジョブを次の走査の再開位置で識別するため、一意性が効く
		// 識別子も同じものになる。
		record.id = a.StartAfter
	default:
		i.t.Fatalf("想定外のジョブが投入されました: %T", args)
	}
	record.skipped = i.skipIDs[record.id]

	i.mu.Lock()
	i.records = append(i.records, record)
	i.mu.Unlock()

	if i.failKinds[record.kind] {
		return nil, errors.New("注入したジョブキューのエラー")
	}
	return &rivertype.JobInsertResult{UniqueSkippedAsDuplicate: record.skipped}, nil
}

// attemptedIDsは、リコンシリエーションが指定種別のジョブを投入しようとした
// 識別子を返す。一意性によるskipが返されたものも含む。
func (i *exportJobInserter) attemptedIDs(kind string) []string {
	i.mu.Lock()
	defer i.mu.Unlock()

	ids := make([]string, 0, len(i.records))
	for _, record := range i.records {
		if record.kind == kind {
			ids = append(ids, record.id)
		}
	}
	return ids
}

// insertedIDsは、今回の実行が実際にジョブを投入した識別子を返す。系統ごとの
// 予算が数えるのはこれである。
func (i *exportJobInserter) insertedIDs(kind string) []string {
	i.mu.Lock()
	defer i.mu.Unlock()

	ids := make([]string, 0, len(i.records))
	for _, record := range i.records {
		if record.kind == kind && !record.skipped {
			ids = append(ids, record.id)
		}
	}
	return ids
}

// newExportOnNewTargetは専用の対象を作り、その上にエクスポートを1件作成する。
// 部分ユニークインデックスは1プロフィールにつきqueued / startedを1件しか許さない
// ため、進行中のエクスポートを複数必要とするテストはエクスポートごとにプロフィールを
// 分ける必要がある。
func newExportOnNewTarget(t *testing.T, tx *sql.Tx, status model.ExportStatus, createdAt time.Time) model.ExportID {
	t.Helper()

	target := testutil.NewProfileOwner(t, tx)
	return testutil.NewExportBuilder(t, tx).
		WithProfileID(target.ProfileID).
		WithActorID(target.ActorID).
		WithStatus(status).
		WithCreatedAt(createdAt).
		Build()
}

// reconcileExportsFixtureは、テスト用トランザクションに配線した
// リコンシリエーションのUseCaseと、検証で行を読み直すrepository、およびジョブ
// キューの代役。
type reconcileExportsFixture struct {
	uc               *usecase.ReconcileExportsUsecase
	exportRepo       *repository.ExportRepository
	notificationRepo *repository.ExportCompletionNotificationRepository
	inserter         *exportJobInserter
}

func newReconcileExportsFixture(t *testing.T, tx *sql.Tx, limits usecase.ExportRecoveryLimits) *reconcileExportsFixture {
	t.Helper()

	queries := testutil.QueriesWithTx(tx)
	exportRepo := repository.NewExportRepository(queries)
	notificationRepo := repository.NewExportCompletionNotificationRepository(queries)
	inserter := newExportJobInserter(t)
	return &reconcileExportsFixture{
		uc: usecase.NewReconcileExportsUsecase(
			exportRepo,
			notificationRepo,
			dispatcher.NewDispatcher(inserter),
			limits,
		),
		exportRepo:       exportRepo,
		notificationRepo: notificationRepo,
		inserter:         inserter,
	}
}

func (f *reconcileExportsFixture) reload(t *testing.T, id model.ExportID) *model.Export {
	t.Helper()

	export, err := f.exportRepo.FindByID(context.Background(), id)
	if err != nil {
		t.Fatalf("エクスポートの取得に失敗: %v", err)
	}
	if export == nil {
		t.Fatalf("エクスポートが見つかりません: %v", id)
	}
	return export
}

// assertJobForは、その識別子に対して指定種別のジョブが投入されていなければ
// 失敗する。
func assertJobFor(t *testing.T, inserter *exportJobInserter, kind, id string) {
	t.Helper()
	if !slices.Contains(inserter.insertedIDs(kind), id) {
		t.Errorf("%sジョブが%sに対して投入されていません (投入済み: %v)", kind, id, inserter.insertedIDs(kind))
	}
}

// assertNoJobForは、その識別子に対して指定種別のジョブが試みられていた場合に
// 失敗する。
func assertNoJobFor(t *testing.T, inserter *exportJobInserter, kind, id string) {
	t.Helper()
	if slices.Contains(inserter.attemptedIDs(kind), id) {
		t.Errorf("%sジョブが%sに対して投入されました (投入すべきではありません)", kind, id)
	}
}

// TestReconcileExportsUsecase_ConvergesAbandonedWorkは、プロセス停止により
// durable workだけが残りジョブが無くなる地点を対象とする。生成投入前にcommit
// されたexport、実行中に強制終了された試行、ジョブ未投入の送信待ち完了通知である。
// exportの処理はexportsから、メール処理は独立したoutboxから再導出する。
func TestReconcileExportsUsecase_ConvergesAbandonedWork(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	limits := usecase.DefaultExportRecoveryLimits()

	// ジョブを投入する前にプロセスが終了し、行だけがcommitされた場合。
	// この生成をキューへ入れるものは他に無い。
	t.Run("Riverへの投入前に終了したqueuedの生成ジョブを再投入する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTxRepeatableRead(t)
		fixture := newReconcileExportsFixture(t, tx, limits)

		abandoned := newExportOnNewTarget(t, tx, model.ExportStatusQueued, time.Now().Add(-2*time.Hour))
		// 作成直後のもの。即時投入がまだ実行中の可能性があり、いま回復しても
		// 通常経路が行っている処理を重ねるだけになる。
		fresh := newExportOnNewTarget(t, tx, model.ExportStatusQueued, time.Now())

		if err := fixture.uc.Execute(ctx); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		assertJobFor(t, fixture.inserter, "generate_export", abandoned.String())
		assertNoJobFor(t, fixture.inserter, "generate_export", fresh.String())
		if got := fixture.reload(t, abandoned); got.Status != model.ExportStatusQueued {
			t.Errorf("got.Status = %v、期待値 = %v", got.Status, model.ExportStatusQueued)
		}
	})

	// 削除中プロフィールのqueued行。生成は削除マーカーで止まるため、ここで投入
	// したジョブは行に触れずに戻る。マーカーは戻らないので、この行を回復対象にすると、
	// 削除が終わらない限り同じジョブを投入し続けることになる。
	t.Run("削除が始まったプロフィールのqueuedは再投入しない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTxRepeatableRead(t)
		fixture := newReconcileExportsFixture(t, tx, limits)

		deletingTarget := testutil.NewProfileOwner(t, tx)
		deleting := testutil.NewExportBuilder(t, tx).
			WithProfileID(deletingTarget.ProfileID).
			WithActorID(deletingTarget.ActorID).
			WithStatus(model.ExportStatusQueued).
			WithCreatedAt(time.Now().Add(-2 * time.Hour)).
			Build()
		if _, err := tx.Exec(
			"UPDATE profiles SET export_deletion_started_at = NOW() WHERE id = $1",
			uuid.UUID(deletingTarget.ProfileID),
		); err != nil {
			t.Fatalf("プロフィール削除開始の記録に失敗: %v", err)
		}

		if err := fixture.uc.Execute(ctx); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		assertNoJobFor(t, fixture.inserter, "generate_export", deleting.String())
		if got := fixture.reload(t, deleting); got.Status != model.ExportStatusQueued {
			t.Errorf("got.Status = %v、期待値 = %v", got.Status, model.ExportStatusQueued)
		}
	})

	// panic・timeout・SIGKILLによって何も記録せずに終わった試行。行はstartedの
	// まま試行が残っているため、諦めるのではなくキューへ戻す。
	t.Run("放置されたstartedをqueuedへ戻して再投入する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTxRepeatableRead(t)
		fixture := newReconcileExportsFixture(t, tx, limits)

		abandoned := newExportOnNewTarget(t, tx, model.ExportStatusStarted, time.Now().Add(-2*time.Hour))
		// 開始したばかりのもの。試行はまだ生成のtimeoutの内側にあり、完了する
		// ことを許されている。
		running := newExportOnNewTarget(t, tx, model.ExportStatusStarted, time.Now())

		if err := fixture.uc.Execute(ctx); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		assertJobFor(t, fixture.inserter, "generate_export", abandoned.String())
		assertNoJobFor(t, fixture.inserter, "generate_export", running.String())

		got := fixture.reload(t, abandoned)
		if got.Status != model.ExportStatusQueued {
			t.Errorf("got.Status = %v、期待値 = %v", got.Status, model.ExportStatusQueued)
		}
		// queuedの状態チェックを満たすようstarted_atはクリアされ、
		// attempt_countは残る。次回の回復が上限判定に数えるのがこの値であるため。
		if got.StartedAt != nil {
			t.Errorf("got.StartedAt = %v、期待値 = nil", got.StartedAt)
		}

		if reloaded := fixture.reload(t, running); reloaded.Status != model.ExportStatusStarted {
			t.Errorf("実行中のエクスポートのstatus = %v、期待値 = %v", reloaded.Status, model.ExportStatusStarted)
		}
	})

	// 同じ放置された試行でも、削除中のプロフィールの場合。生成は削除マーカーで
	// 止まるため、行を差し戻してもその実行の予算を、行に触れずに戻るジョブへ使うだけに
	// なる。行を収束させるのは親削除の仕事である。
	t.Run("削除が始まったプロフィールのstartedは差し戻さない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTxRepeatableRead(t)
		fixture := newReconcileExportsFixture(t, tx, limits)

		deletingTarget := testutil.NewProfileOwner(t, tx)
		deleting := testutil.NewExportBuilder(t, tx).
			WithProfileID(deletingTarget.ProfileID).
			WithActorID(deletingTarget.ActorID).
			WithStatus(model.ExportStatusStarted).
			WithCreatedAt(time.Now().Add(-2 * time.Hour)).
			Build()
		if _, err := tx.Exec(
			"UPDATE profiles SET export_deletion_started_at = NOW() WHERE id = $1",
			uuid.UUID(deletingTarget.ProfileID),
		); err != nil {
			t.Fatalf("プロフィール削除開始の記録に失敗: %v", err)
		}

		if err := fixture.uc.Execute(ctx); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		assertNoJobFor(t, fixture.inserter, "generate_export", deleting.String())
		if got := fixture.reload(t, deleting); got.Status != model.ExportStatusStarted {
			t.Errorf("got.Status = %v、期待値 = %v", got.Status, model.ExportStatusStarted)
		}
	})

	// 同じ中断でも、試行を使い切ったエクスポートの場合。startedのまま残すと
	// ユーザーには終わらない生成が表示されるため、再実行できる失敗として閉じる。
	t.Run("試行を使い切ったstartedをfailedへ収束させる", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTxRepeatableRead(t)
		fixture := newReconcileExportsFixture(t, tx, limits)

		target := testutil.NewProfileOwner(t, tx)
		exhausted := testutil.NewExportBuilder(t, tx).
			WithProfileID(target.ProfileID).
			WithActorID(target.ActorID).
			WithStatus(model.ExportStatusStarted).
			WithAttemptCount(dispatcher.GenerateExportMaxAttempts).
			WithCreatedAt(time.Now().Add(-2 * time.Hour)).
			Build()

		if err := fixture.uc.Execute(ctx); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		got := fixture.reload(t, exhausted)
		if got.Status != model.ExportStatusFailed {
			t.Errorf("got.Status = %v、期待値 = %v", got.Status, model.ExportStatusFailed)
		}
		if got.FinishedAt == nil {
			t.Error("got.FinishedAt = nil、時刻を期待")
		}
		assertNoJobFor(t, fixture.inserter, "generate_export", exhausted.String())
	})

	// 送信待ち通知のcommit後、ジョブ投入前にプロセスが終了した場合。outbox行
	// だけからメールを回復できる。
	t.Run("送信待ちoutboxの完了通知を投入する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTxRepeatableRead(t)
		fixture := newReconcileExportsFixture(t, tx, limits)

		pendingTarget := testutil.NewProfileOwner(t, tx)
		pending := testutil.NewExportBuilder(t, tx).
			WithProfileID(pendingTarget.ProfileID).
			WithActorID(pendingTarget.ActorID).
			WithStatus(model.ExportStatusSucceeded).
			WithCreatedAt(time.Now().Add(-time.Hour)).
			Build()
		testutil.NewExportCompletionNotificationBuilder(t, tx).
			WithExportID(pending).
			WithActorID(pendingTarget.ActorID).
			WithCreatedAt(time.Now().Add(-time.Hour)).
			Build()

		withoutOutbox := newExportOnNewTarget(t, tx, model.ExportStatusSucceeded, time.Now().Add(-time.Hour))

		if err := fixture.uc.Execute(ctx); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		assertJobFor(t, fixture.inserter, "send_export_completed_email", pending.String())
		assertNoJobFor(t, fixture.inserter, "send_export_completed_email", withoutOutbox.String())
	})

	// 削除中プロフィールの送信待ち通知。配信は削除マーカーで止まるため、ここで
	// 投入したジョブは送信せずに戻る。マーカーは戻らないので、この行を回復対象にすると、
	// 削除が終わらない限り同じジョブを投入し、その実行の予算を消費し続ける。通知を
	// 取り消すのは親削除の仕事である。
	t.Run("削除が始まったプロフィールの送信待ち通知は再投入しない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTxRepeatableRead(t)
		fixture := newReconcileExportsFixture(t, tx, limits)

		deletingTarget := testutil.NewProfileOwner(t, tx)
		deleting := testutil.NewExportBuilder(t, tx).
			WithProfileID(deletingTarget.ProfileID).
			WithActorID(deletingTarget.ActorID).
			WithStatus(model.ExportStatusSucceeded).
			WithCreatedAt(time.Now().Add(-time.Hour)).
			Build()
		testutil.NewExportCompletionNotificationBuilder(t, tx).
			WithExportID(deleting).
			WithActorID(deletingTarget.ActorID).
			WithCreatedAt(time.Now().Add(-time.Hour)).
			Build()
		if _, err := tx.Exec(
			"UPDATE profiles SET export_deletion_started_at = NOW() WHERE id = $1",
			uuid.UUID(deletingTarget.ProfileID),
		); err != nil {
			t.Fatalf("プロフィール削除開始の記録に失敗: %v", err)
		}

		if err := fixture.uc.Execute(ctx); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		assertNoJobFor(t, fixture.inserter, "send_export_completed_email", deleting.String())
		got, err := fixture.notificationRepo.FindByExportID(ctx, deleting)
		if err != nil {
			t.Fatalf("FindByExportID()のエラー = %v", err)
		}
		if got == nil {
			t.Error("送信待ち通知が失われている")
		}
	})

	// cleanupの投入が失われたまま成功が記録された場合。古いアーカイブが
	// オブジェクトストレージに残り続ける。本機能が繰り返さないと決めた保持の失敗が
	// これである。
	t.Run("旧succeededを持つプロフィールの削除ジョブを投入する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTxRepeatableRead(t)
		fixture := newReconcileExportsFixture(t, tx, limits)

		withOld := testutil.NewProfileOwner(t, tx)
		for _, createdAt := range []time.Time{time.Now().Add(-2 * time.Hour), time.Now().Add(-time.Hour)} {
			testutil.NewExportBuilder(t, tx).
				WithProfileID(withOld.ProfileID).
				WithActorID(withOld.ActorID).
				WithStatus(model.ExportStatusSucceeded).
				WithCreatedAt(createdAt).
				Build()
		}

		// 成功が1件だけの状態は保持ポリシー上の定常状態で、削除すべき古いものが
		// 無い。
		withLatestOnly := testutil.NewProfileOwner(t, tx)
		testutil.NewExportBuilder(t, tx).
			WithProfileID(withLatestOnly.ProfileID).
			WithActorID(withLatestOnly.ActorID).
			WithStatus(model.ExportStatusSucceeded).
			WithCreatedAt(time.Now().Add(-time.Hour)).
			Build()

		if err := fixture.uc.Execute(ctx); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		assertJobFor(t, fixture.inserter, "cleanup_old_exports", withOld.ProfileID.String())
		assertNoJobFor(t, fixture.inserter, "cleanup_old_exports", withLatestOnly.ProfileID.String())
	})
}

// TestReconcileExportsUsecase_BudgetCountsOnlyNewWorkは、系統ごとの予算が何を
// 制限するかを固定する。ジョブがすでに存在する候補は予算を使わずに走査されるため、
// 先頭がすでに処理中のバックログが、その後ろの候補への到達を止めることはない。固定順序
// への単純なLIMITが生む失敗がこれである。
func TestReconcileExportsUsecase_BudgetCountsOnlyNewWork(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("skipされた候補は予算を使わず後続の候補に到達する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTxRepeatableRead(t)
		// 1ページ1候補・1回の実行で新たに引き受けるのも1候補とする。skipした
		// 先頭を数えない場合にのみ、この実行は2件目に到達できる。
		fixture := newReconcileExportsFixture(t, tx, usecase.ExportRecoveryLimits{PageSize: 1, Budget: 1})

		// 最も古い候補が先頭に並ぶため、走査はこれを越える必要がある。
		alreadyQueued := newExportOnNewTarget(t, tx, model.ExportStatusQueued, time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC))
		behind := newExportOnNewTarget(t, tx, model.ExportStatusQueued, time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC))
		fixture.inserter.skipIDs[alreadyQueued.String()] = true

		if err := fixture.uc.Execute(ctx); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		if !slices.Contains(fixture.inserter.attemptedIDs("generate_export"), alreadyQueued.String()) {
			t.Errorf("先頭の候補%sに対する投入が試みられていません", alreadyQueued.String())
		}
		assertJobFor(t, fixture.inserter, "generate_export", behind.String())
	})

	t.Run("予算に達したら残りの候補は次回の実行に回す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTxRepeatableRead(t)
		fixture := newReconcileExportsFixture(t, tx, usecase.ExportRecoveryLimits{PageSize: 1, Budget: 1})

		first := newExportOnNewTarget(t, tx, model.ExportStatusQueued, time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC))
		second := newExportOnNewTarget(t, tx, model.ExportStatusQueued, time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC))

		if err := fixture.uc.Execute(ctx); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		assertJobFor(t, fixture.inserter, "generate_export", first.String())
		assertNoJobFor(t, fixture.inserter, "generate_export", second.String())
	})
}

// TestReconcileExportsUsecase_ContinuesAfterStreamFailureは、各系統が独立に
// 回復することを固定する。各系統は別々の行から別々の処理を再構築するため、ある種別の
// ジョブを受け付けないキューによって、他の系統が行えた回復まで失われてはならない。
// 失敗はそれでも報告され、次回の定期実行が再試行にあたる。
func TestReconcileExportsUsecase_ContinuesAfterStreamFailure(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, tx := testutil.SetupTxRepeatableRead(t)
	fixture := newReconcileExportsFixture(t, tx, usecase.DefaultExportRecoveryLimits())
	fixture.inserter.failKinds["generate_export"] = true

	newExportOnNewTarget(t, tx, model.ExportStatusQueued, time.Now().Add(-2*time.Hour))
	pendingTarget := testutil.NewProfileOwner(t, tx)
	pending := testutil.NewExportBuilder(t, tx).
		WithProfileID(pendingTarget.ProfileID).
		WithActorID(pendingTarget.ActorID).
		WithStatus(model.ExportStatusSucceeded).
		WithCreatedAt(time.Now().Add(-time.Hour)).
		Build()
	testutil.NewExportCompletionNotificationBuilder(t, tx).
		WithExportID(pending).
		WithActorID(pendingTarget.ActorID).
		WithCreatedAt(time.Now().Add(-time.Hour)).
		Build()

	err := fixture.uc.Execute(ctx)
	if err == nil {
		t.Fatal("Execute()のエラー = nil、エラーを期待")
	}
	if !strings.Contains(err.Error(), "queuedエクスポートの回復に失敗") {
		t.Errorf("Execute()のエラー = %v、queuedエクスポートの回復の失敗を報告することを期待", err)
	}
	assertJobFor(t, fixture.inserter, "send_export_completed_email", pending.String())
}
