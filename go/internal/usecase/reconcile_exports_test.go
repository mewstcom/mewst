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

// jobRecord is one insert reconciliation attempted, with the identifier the job
// carries and whether River answered with the unique skip it uses when a job
// for the same work intent is still outstanding.
//
// [Ja] jobRecord はリコンシリエーションが試みた投入 1 件。ジョブが持つ識別子と、
// 同じ作業依頼のジョブが未完了のときに River が返す一意性による skip だったかを持つ。
type jobRecord struct {
	kind    string
	id      string
	skipped bool
}

// exportJobInserter stands in for the job queue of the export recovery jobs. It
// records every attempted insert so a test can tell the candidates a run walked
// past from the ones it took on, and it can answer with the unique skip or an
// error to put the caller in the states only a busy or broken queue produces.
//
// [Ja] exportJobInserter はエクスポートの回復系ジョブにおけるジョブキューの代役。
// 試みられた投入をすべて記録し、実行が走査しただけの候補と引き受けた候補をテストが
// 区別できるようにする。また、一意性による skip やエラーを返すことで、混み合ったキューや
// 壊れたキューでしか生じない状態を作れる。
type exportJobInserter struct {
	t *testing.T

	// skipIDs are the identifiers answered with the unique skip, standing for a
	// job for that work intent already waiting or running.
	//
	// [Ja] skipIDs は一意性による skip を返す識別子。その作業依頼のジョブがすでに
	// 待機中または実行中であることを表す。
	skipIDs map[string]bool

	// failKinds are the job kinds the queue refuses, standing for a queue that is
	// unreachable for that kind of work.
	//
	// [Ja] failKinds はキューが受け付けないジョブ種別。その種類の処理についてキューへ
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
		// The sweep identifies its continuation by where the next walk resumes,
		// so that is the identifier the uniqueness applies to as well.
		//
		// [Ja] 掃除は継続ジョブを次の走査の再開位置で識別するため、一意性が効く
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

// attemptedIDs returns the identifiers reconciliation tried to insert a job of
// the given kind for, including the ones the unique skip answered.
//
// [Ja] attemptedIDs は、リコンシリエーションが指定種別のジョブを投入しようとした
// 識別子を返す。一意性による skip が返されたものも含む。
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

// insertedIDs returns the identifiers whose job this run actually inserted,
// which is what the per-stream budget counts.
//
// [Ja] insertedIDs は、今回の実行が実際にジョブを投入した識別子を返す。系統ごとの
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

// newExportOnNewTarget creates one export on a target of its own. The partial
// unique index allows a profile only one queued or started export, so tests
// that need several in-progress exports need a profile per export.
//
// [Ja] newExportOnNewTarget は専用の対象を作り、その上にエクスポートを 1 件作成する。
// 部分ユニークインデックスは 1 プロフィールにつき queued / started を 1 件しか許さない
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

// reconcileExportsFixture is the reconciliation UseCase wired to a test
// transaction, together with the repository the assertions read rows back with
// and the job queue stand-in.
//
// [Ja] reconcileExportsFixture は、テスト用トランザクションに配線した
// リコンシリエーションの UseCase と、検証で行を読み直す repository、およびジョブ
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

// assertJobFor fails unless a job of the kind was inserted for the identifier.
//
// [Ja] assertJobFor は、その識別子に対して指定種別のジョブが投入されていなければ
// 失敗する。
func assertJobFor(t *testing.T, inserter *exportJobInserter, kind, id string) {
	t.Helper()
	if !slices.Contains(inserter.insertedIDs(kind), id) {
		t.Errorf("%s ジョブが %s に対して投入されていません (投入済み: %v)", kind, id, inserter.insertedIDs(kind))
	}
}

// assertNoJobFor fails when a job of the kind was attempted for the identifier.
//
// [Ja] assertNoJobFor は、その識別子に対して指定種別のジョブが試みられていた場合に
// 失敗する。
func assertNoJobFor(t *testing.T, inserter *exportJobInserter, kind, id string) {
	t.Helper()
	if slices.Contains(inserter.attemptedIDs(kind), id) {
		t.Errorf("%s ジョブが %s に対して投入されました (投入すべきではありません)", kind, id)
	}
}

// TestReconcileExportsUsecase_ConvergesAbandonedWork covers the points a
// process can stop at and leave durable work without a job: an export committed
// before generation was enqueued, an attempt killed while running, and a
// pending completion notification whose job was never enqueued. Export work is
// re-derived from exports and email work from its independent outbox.
//
// [Ja] TestReconcileExportsUsecase_ConvergesAbandonedWork は、プロセス停止により
// durable work だけが残りジョブが無くなる地点を対象とする。生成投入前に commit
// された export、実行中に強制終了された試行、ジョブ未投入の送信待ち完了通知である。
// export の処理は exports から、メール処理は独立した outbox から再導出する。
func TestReconcileExportsUsecase_ConvergesAbandonedWork(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	limits := usecase.DefaultExportRecoveryLimits()

	// The row committed while the process died before inserting its job: nothing
	// else will ever queue this generation.
	//
	// [Ja] ジョブを投入する前にプロセスが終了し、行だけが commit された場合。
	// この生成をキューへ入れるものは他に無い。
	t.Run("River への投入前に終了した queued の生成ジョブを再投入する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTxRepeatableRead(t)
		fixture := newReconcileExportsFixture(t, tx, limits)

		abandoned := newExportOnNewTarget(t, tx, model.ExportStatusQueued, time.Now().Add(-2*time.Hour))
		// Just created: its immediate insert may still be in flight, so recovering
		// it now would only duplicate work the normal path is already doing.
		//
		// [Ja] 作成直後のもの。即時投入がまだ実行中の可能性があり、いま回復しても
		// 通常経路が行っている処理を重ねるだけになる。
		fresh := newExportOnNewTarget(t, tx, model.ExportStatusQueued, time.Now())

		if err := fixture.uc.Execute(ctx); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		assertJobFor(t, fixture.inserter, "generate_export", abandoned.String())
		assertNoJobFor(t, fixture.inserter, "generate_export", fresh.String())
		if got := fixture.reload(t, abandoned); got.Status != model.ExportStatusQueued {
			t.Errorf("got.Status = %v, want %v", got.Status, model.ExportStatusQueued)
		}
	})

	// The queued row of a profile being deleted: generation stops at the deletion
	// marker, so a job inserted here would return without touching the row. The
	// marker is never cleared, so recovering the row would keep inserting that
	// job for as long as the deletion has not finished.
	//
	// [Ja] 削除中プロフィールの queued 行。生成は削除マーカーで止まるため、ここで投入
	// したジョブは行に触れずに戻る。マーカーは戻らないので、この行を回復対象にすると、
	// 削除が終わらない限り同じジョブを投入し続けることになる。
	t.Run("削除が始まったプロフィールの queued は再投入しない", func(t *testing.T) {
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
			t.Fatalf("Execute() error = %v", err)
		}

		assertNoJobFor(t, fixture.inserter, "generate_export", deleting.String())
		if got := fixture.reload(t, deleting); got.Status != model.ExportStatusQueued {
			t.Errorf("got.Status = %v, want %v", got.Status, model.ExportStatusQueued)
		}
	})

	// The attempt that a panic, a timeout or a SIGKILL ended without recording
	// anything: the row stays started with attempts left, so the export is put
	// back in the queue rather than given up on.
	//
	// [Ja] panic・timeout・SIGKILL によって何も記録せずに終わった試行。行は started の
	// まま試行が残っているため、諦めるのではなくキューへ戻す。
	t.Run("放置された started を queued へ戻して再投入する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTxRepeatableRead(t)
		fixture := newReconcileExportsFixture(t, tx, limits)

		abandoned := newExportOnNewTarget(t, tx, model.ExportStatusStarted, time.Now().Add(-2*time.Hour))
		// Started moments ago: the attempt is still inside the generation timeout
		// and is allowed to finish.
		//
		// [Ja] 開始したばかりのもの。試行はまだ生成の timeout の内側にあり、完了する
		// ことを許されている。
		running := newExportOnNewTarget(t, tx, model.ExportStatusStarted, time.Now())

		if err := fixture.uc.Execute(ctx); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		assertJobFor(t, fixture.inserter, "generate_export", abandoned.String())
		assertNoJobFor(t, fixture.inserter, "generate_export", running.String())

		got := fixture.reload(t, abandoned)
		if got.Status != model.ExportStatusQueued {
			t.Errorf("got.Status = %v, want %v", got.Status, model.ExportStatusQueued)
		}
		// started_at is cleared so the queued state check holds, while
		// attempt_count survives: it is what the next recovery counts towards the
		// limit.
		//
		// [Ja] queued の状態チェックを満たすよう started_at はクリアされ、
		// attempt_count は残る。次回の回復が上限判定に数えるのがこの値であるため。
		if got.StartedAt != nil {
			t.Errorf("got.StartedAt = %v, want nil", got.StartedAt)
		}

		if reloaded := fixture.reload(t, running); reloaded.Status != model.ExportStatusStarted {
			t.Errorf("実行中のエクスポートの status = %v, want %v", reloaded.Status, model.ExportStatusStarted)
		}
	})

	// The same abandoned attempt, but on a profile being deleted: generation
	// stops at the deletion marker, so requeueing the row would spend this run's
	// budget on a job that returns without touching it. Converging the row is
	// profile deletion's work.
	//
	// [Ja] 同じ放置された試行でも、削除中のプロフィールの場合。生成は削除マーカーで
	// 止まるため、行を差し戻してもその実行の予算を、行に触れずに戻るジョブへ使うだけに
	// なる。行を収束させるのは親削除の仕事である。
	t.Run("削除が始まったプロフィールの started は差し戻さない", func(t *testing.T) {
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
			t.Fatalf("Execute() error = %v", err)
		}

		assertNoJobFor(t, fixture.inserter, "generate_export", deleting.String())
		if got := fixture.reload(t, deleting); got.Status != model.ExportStatusStarted {
			t.Errorf("got.Status = %v, want %v", got.Status, model.ExportStatusStarted)
		}
	})

	// The same interruption, but on an export that has used up its attempts:
	// leaving it started would show the user a generation that never ends, so it
	// is closed as a failure they can retry from.
	//
	// [Ja] 同じ中断でも、試行を使い切ったエクスポートの場合。started のまま残すと
	// ユーザーには終わらない生成が表示されるため、再実行できる失敗として閉じる。
	t.Run("試行を使い切った started を failed へ収束させる", func(t *testing.T) {
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
			t.Fatalf("Execute() error = %v", err)
		}

		got := fixture.reload(t, exhausted)
		if got.Status != model.ExportStatusFailed {
			t.Errorf("got.Status = %v, want %v", got.Status, model.ExportStatusFailed)
		}
		if got.FinishedAt == nil {
			t.Error("got.FinishedAt = nil, want a time")
		}
		assertNoJobFor(t, fixture.inserter, "generate_export", exhausted.String())
	})

	// The process stopped after the pending notification was committed but
	// before its job was enqueued. The outbox row is enough to recover the email.
	//
	// [Ja] 送信待ち通知の commit 後、ジョブ投入前にプロセスが終了した場合。outbox 行
	// だけからメールを回復できる。
	t.Run("送信待ち outbox の完了通知を投入する", func(t *testing.T) {
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
			t.Fatalf("Execute() error = %v", err)
		}

		assertJobFor(t, fixture.inserter, "send_export_completed_email", pending.String())
		assertNoJobFor(t, fixture.inserter, "send_export_completed_email", withoutOutbox.String())
	})

	// The pending notification of a profile being deleted: delivery stops at the
	// deletion marker, so a job inserted here would return without sending. The
	// marker is never cleared, so recovering the row would keep inserting that
	// job, and spending this run's budget on it, for as long as the deletion has
	// not finished. Cancelling the notification is profile deletion's work.
	//
	// [Ja] 削除中プロフィールの送信待ち通知。配信は削除マーカーで止まるため、ここで
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
			t.Fatalf("Execute() error = %v", err)
		}

		assertNoJobFor(t, fixture.inserter, "send_export_completed_email", deleting.String())
		got, err := fixture.notificationRepo.FindByExportID(ctx, deleting)
		if err != nil {
			t.Fatalf("FindByExportID() error = %v", err)
		}
		if got == nil {
			t.Error("送信待ち通知が失われている")
		}
	})

	// The success recorded while the cleanup insert was lost: the old archive
	// stays in the object storage, which is the retention failure this feature
	// set out not to repeat.
	//
	// [Ja] cleanup の投入が失われたまま成功が記録された場合。古いアーカイブが
	// オブジェクトストレージに残り続ける。本機能が繰り返さないと決めた保持の失敗が
	// これである。
	t.Run("旧 succeeded を持つプロフィールの削除ジョブを投入する", func(t *testing.T) {
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

		// A single success is the steady state of the retention policy: there is
		// nothing older to delete.
		//
		// [Ja] 成功が 1 件だけの状態は保持ポリシー上の定常状態で、削除すべき古いものが
		// 無い。
		withLatestOnly := testutil.NewProfileOwner(t, tx)
		testutil.NewExportBuilder(t, tx).
			WithProfileID(withLatestOnly.ProfileID).
			WithActorID(withLatestOnly.ActorID).
			WithStatus(model.ExportStatusSucceeded).
			WithCreatedAt(time.Now().Add(-time.Hour)).
			Build()

		if err := fixture.uc.Execute(ctx); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		assertJobFor(t, fixture.inserter, "cleanup_old_exports", withOld.ProfileID.String())
		assertNoJobFor(t, fixture.inserter, "cleanup_old_exports", withLatestOnly.ProfileID.String())
	})
}

// TestReconcileExportsUsecase_BudgetCountsOnlyNewWork pins what the per-stream
// budget limits. A candidate whose job already exists is walked past without
// spending budget, so a backlog whose head is already being worked on cannot
// stop the run from reaching the candidates behind it — the failure a plain
// LIMIT over a fixed order produces.
//
// [Ja] TestReconcileExportsUsecase_BudgetCountsOnlyNewWork は、系統ごとの予算が何を
// 制限するかを固定する。ジョブがすでに存在する候補は予算を使わずに走査されるため、
// 先頭がすでに処理中のバックログが、その後ろの候補への到達を止めることはない。固定順序
// への単純な LIMIT が生む失敗がこれである。
func TestReconcileExportsUsecase_BudgetCountsOnlyNewWork(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("skip された候補は予算を使わず後続の候補に到達する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTxRepeatableRead(t)
		// One candidate per page and one new candidate per run: the run can only
		// reach the second candidate by not counting the skipped first one.
		//
		// [Ja] 1 ページ 1 候補・1 回の実行で新たに引き受けるのも 1 候補とする。skip した
		// 先頭を数えない場合にのみ、この実行は 2 件目に到達できる。
		fixture := newReconcileExportsFixture(t, tx, usecase.ExportRecoveryLimits{PageSize: 1, Budget: 1})

		// The oldest candidate sorts first, so it is the head the walk has to get
		// past.
		//
		// [Ja] 最も古い候補が先頭に並ぶため、走査はこれを越える必要がある。
		alreadyQueued := newExportOnNewTarget(t, tx, model.ExportStatusQueued, time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC))
		behind := newExportOnNewTarget(t, tx, model.ExportStatusQueued, time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC))
		fixture.inserter.skipIDs[alreadyQueued.String()] = true

		if err := fixture.uc.Execute(ctx); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if !slices.Contains(fixture.inserter.attemptedIDs("generate_export"), alreadyQueued.String()) {
			t.Errorf("先頭の候補 %s に対する投入が試みられていません", alreadyQueued.String())
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
			t.Fatalf("Execute() error = %v", err)
		}

		assertJobFor(t, fixture.inserter, "generate_export", first.String())
		assertNoJobFor(t, fixture.inserter, "generate_export", second.String())
	})
}

// TestReconcileExportsUsecase_ContinuesAfterStreamFailure pins that the streams
// recover independently. They rebuild different work from different rows, so a
// queue that refuses one kind of job must not cost the run the recoveries the
// other streams can still make; the failure is still reported, and the next
// scheduled run is the retry.
//
// [Ja] TestReconcileExportsUsecase_ContinuesAfterStreamFailure は、各系統が独立に
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
		t.Fatal("Execute() = nil, want an error")
	}
	if !strings.Contains(err.Error(), "queued エクスポートの回復に失敗") {
		t.Errorf("Execute() error = %v, want it to report the queued stream", err)
	}
	assertJobFor(t, fixture.inserter, "send_export_completed_email", pending.String())
}
