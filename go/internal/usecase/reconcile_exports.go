package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

const (
	// ReconcileExportsTimeout bounds one reconciliation run. The work per run is
	// already bounded by the page size and the per-stream budget, so this is the
	// backstop for a run that stopped making progress rather than a limit the
	// normal run approaches.
	//
	// It is kept strictly below the interval the reconciliation is scheduled at.
	// The job is unique across the running state, so a run still holding its
	// worker when the next periodic insert arrives makes that insert skip, and
	// the recovery a stalled run already delayed would be delayed by another
	// interval. The interval lives in the worker package, which is where a test
	// pins the relation.
	//
	// [Ja] ReconcileExportsTimeout はリコンシリエーション 1 回の実行の上限。1 回の
	// 実行の仕事量はページサイズと系統ごとの予算ですでに有界なので、これは通常の実行が
	// 近づく上限ではなく、前進しなくなった実行に対する歯止め。
	//
	// 値はリコンシリエーションの定期実行間隔より厳密に小さく保つ。このジョブは running を
	// 含む状態で一意なため、次の定期投入の時点でまだ worker を保持している実行があると
	// その投入が skip され、停滞した実行がすでに遅らせた回復がもう 1 間隔分遅れる。間隔は
	// worker パッケージにあり、この関係は同パッケージのテストで固定する。
	ReconcileExportsTimeout = 4 * time.Minute

	// exportQueuedGrace is how long a queued export is left alone before
	// reconciliation re-enqueues its generation job. The gap this closes is
	// between the commit of the export row and the immediate insert that follows
	// it, so it only has to outlast that insert; re-enqueueing is idempotent
	// through the unique job, so a grace shorter than it needs to be costs
	// nothing but a skipped insert.
	//
	// [Ja] exportQueuedGrace は、リコンシリエーションが生成ジョブを再投入するまでに
	// queued のエクスポートを放置する時間。ここで閉じる隙間はエクスポート行の commit と
	// 直後の即時投入の間にあるため、その投入より長ければよい。再投入は一意ジョブに
	// より冪等なので、短すぎても skip される投入が 1 回増えるだけで済む。
	exportQueuedGrace = 1 * time.Minute

	// exportStartedGrace is added to the generation timeout before a started
	// export is treated as stale. The timeout is what the job queue gives one
	// attempt, so the grace only has to cover the worker noticing the timeout and
	// recording the outcome; recovering earlier would fight with an attempt that
	// is still allowed to run.
	//
	// [Ja] exportStartedGrace は、started のエクスポートを停滞と見なすまでに生成の
	// timeout へ上乗せする時間。timeout はジョブキューが 1 回の試行に与える時間なので、
	// 猶予は worker が timeout に気付いて結果を記録するまでを賄えばよい。これより早く
	// 回復させると、まだ実行を許されている試行と競合する。
	exportStartedGrace = 5 * time.Minute

	// exportNotificationGrace is how long a pending completion-notification
	// outbox row waits before reconciliation enqueues its email again. The grace
	// period keeps recovery from racing the normal enqueue and delivery path.
	//
	// [Ja] exportNotificationGrace は、リコンシリエーションが完了通知メールを再投入する
	// まで送信待ち outbox 行を待つ時間。成功直後の投入・配信経路と回復経路が
	// 競合しないための猶予とする。
	exportNotificationGrace = 5 * time.Minute

	// defaultExportRecoveryPageSize is how many candidates one recovery query
	// holds at a time.
	//
	// [Ja] defaultExportRecoveryPageSize は 1 回の回復クエリが一度に保持する候補数。
	defaultExportRecoveryPageSize int32 = 100

	// defaultExportRecoveryBudget is how many candidates one run takes on per
	// stream.
	//
	// [Ja] defaultExportRecoveryBudget は 1 回の実行が系統ごとに引き受ける候補数。
	defaultExportRecoveryBudget = 100
)

// ExportRecoveryLimits bounds one reconciliation run. PageSize limits how many
// candidates a single query holds, and Budget limits how many candidates the
// run takes on per stream, so a backlog is worked off over several runs instead
// of one run loading it all and inserting a burst of jobs.
//
// The two are separate because a candidate the run walks past is not a
// candidate it takes on: a candidate whose unique job already exists is skipped
// without spending budget, and the walk keeps advancing its cursor so a stuck
// head cannot consume every run's budget on its own.
//
// [Ja] ExportRecoveryLimits はリコンシリエーション 1 回の実行を有界にする。PageSize は
// 1 クエリが保持する候補数を、Budget は 1 回の実行が系統ごとに引き受ける候補数を制限
// する。これにより、バックログは 1 回の実行が全件を読み込んでジョブをバーストさせるの
// ではなく、複数回の実行で処理される。
//
// 2 つを分けているのは、走査した候補と引き受けた候補が別物だからである。一意ジョブが
// すでに存在する候補は予算を使わずに skip され、走査は cursor を進め続けるため、停滞
// した先頭候補が毎回の予算を独占することがない。
type ExportRecoveryLimits struct {
	PageSize int32
	Budget   int
}

// DefaultExportRecoveryLimits returns the limits used in production. Values
// that are not positive are replaced with these, so a caller cannot configure a
// reconciliation into silently doing nothing.
//
// [Ja] DefaultExportRecoveryLimits は本番で使う上限値を返す。正でない値はこれで
// 置き換えるため、呼び出し側の設定によってリコンシリエーションが黙って何もしない
// 状態にはならない。
func DefaultExportRecoveryLimits() ExportRecoveryLimits {
	return ExportRecoveryLimits{
		PageSize: defaultExportRecoveryPageSize,
		Budget:   defaultExportRecoveryBudget,
	}
}

// normalized returns the limits with non-positive values replaced by the
// defaults.
//
// [Ja] normalized は正でない値を既定値で置き換えた上限値を返す。
func (l ExportRecoveryLimits) normalized() ExportRecoveryLimits {
	defaults := DefaultExportRecoveryLimits()
	if l.PageSize <= 0 {
		l.PageSize = defaults.PageSize
	}
	if l.Budget <= 0 {
		l.Budget = defaults.Budget
	}
	return l
}

// ReconcileExportsUsecase converges exports that the normal flow left behind:
// an export row committed while its generation job was never inserted, an
// attempt whose process stopped mid-flight, or follow-up work that was never
// enqueued. Generation and cleanup are re-derived from exports; completion
// email work is re-derived from its independent outbox so export retention
// cannot erase it.
//
// It only reads and moves database rows, and enqueues unique jobs. Objects in
// the object storage are not touched here: an export closed as failed releases
// its object, and collecting that object is the orphan sweep's job.
//
// Every job kind it enqueues needs its Worker registered before any export can
// reach the state that produces the candidate. River answers a fetch for an
// unregistered kind with an error, which uses the job's attempts up and
// discards it; discarded is outside the uniqueness set, so the next run inserts
// the same job again and the pair repeats for as long as the candidate exists.
//
// [Ja] ReconcileExportsUsecase は通常の流れが取り残したエクスポートを収束させる。
// 生成ジョブが投入されないまま commit された export 行、処理途中でプロセスが終了した
// 試行、投入されなかった後続処理などが対象。生成と cleanup は exports から、完了
// メールは独立した outbox から再導出するため、export の保持処理が通知を消すことはない。
//
// 本 UseCase は DB の行を読んで動かし、一意ジョブを投入するだけである。オブジェクト
// ストレージには触れない。failed として閉じたエクスポートはオブジェクトを手放し、その
// オブジェクトの回収は孤児回収の担当であるため。
//
// 本 UseCase が投入するジョブ種別は、その候補を生む状態にエクスポートが到達するより前に、
// 対応する Worker が登録されている必要がある。River は未登録の種別の fetch にエラーを
// 返し、ジョブは試行を使い切って discarded になる。discarded は一意性の状態集合の外に
// あるため、次回の実行が同じジョブを投入し直し、候補が存在する限りこの組が繰り返される。
type ReconcileExportsUsecase struct {
	exportRepo       *repository.ExportRepository
	notificationRepo *repository.ExportCompletionNotificationRepository
	dispatcher       *dispatcher.Dispatcher
	limits           ExportRecoveryLimits
}

// NewReconcileExportsUsecase creates a ReconcileExportsUsecase. Pass
// DefaultExportRecoveryLimits() unless the caller has a reason to bound a run
// differently.
//
// [Ja] NewReconcileExportsUsecase は ReconcileExportsUsecase を生成する。1 回の実行を
// 別の上限で区切る理由がなければ DefaultExportRecoveryLimits() を渡す。
func NewReconcileExportsUsecase(
	exportRepo *repository.ExportRepository,
	notificationRepo *repository.ExportCompletionNotificationRepository,
	d *dispatcher.Dispatcher,
	limits ExportRecoveryLimits,
) *ReconcileExportsUsecase {
	return &ReconcileExportsUsecase{
		exportRepo:       exportRepo,
		notificationRepo: notificationRepo,
		dispatcher:       d,
		limits:           limits.normalized(),
	}
}

// Execute walks the four recovery streams and returns the errors they hit
// joined together.
//
// Every stream runs even when an earlier one failed. The streams recover
// different work from different rows, so stopping at the first failure would
// let one broken stream hide the progress the others can still make. The job
// runs with a single attempt, and the next scheduled run is the retry.
//
// [Ja] Execute は 4 系統の回復処理を走査し、そこで発生したエラーをまとめて返す。
//
// 前の系統が失敗しても、後続の系統は実行する。各系統は別々の行から別々の処理を回復
// するため、最初の失敗で止めると、1 つの壊れた系統が他の系統の前進を隠してしまう。
// このジョブの試行回数は 1 回で、次回の定期実行が再試行にあたる。
func (uc *ReconcileExportsUsecase) Execute(ctx context.Context) error {
	// One instant drives every threshold, so the streams describe the same
	// moment even when the run is slow.
	//
	// [Ja] すべてのしきい値を 1 つの時刻から導く。実行に時間がかかっても、各系統が
	// 同じ時点を基準にするようにするため。
	now := time.Now()

	errs := []error{
		uc.requeueStaleQueued(ctx, now),
		uc.recoverStaleStarted(ctx, now),
		uc.notifyPendingCompletions(ctx, now),
		uc.cleanupProfilesWithOldSucceeded(ctx),
	}
	return errors.Join(errs...)
}

// requeueStaleQueued re-enqueues the generation job of every export that has
// been queued for longer than the grace, recovering the exports whose immediate
// insert was lost.
//
// [Ja] requeueStaleQueued は、猶予時間より長く queued のままのエクスポートについて
// 生成ジョブを再投入し、即時投入が失われたエクスポートを回復する。
func (uc *ReconcileExportsUsecase) requeueStaleQueued(ctx context.Context, now time.Time) error {
	threshold := now.Add(-exportQueuedGrace)

	taken, err := takeExportRecoveryCandidates(ctx, uc.limits,
		func(ctx context.Context, cursor *repository.ExportRecoveryCursor, pageSize int32) ([]*model.Export, *repository.ExportRecoveryCursor, error) {
			return uc.exportRepo.ListStaleQueued(ctx, threshold, cursor, pageSize)
		},
		func(ctx context.Context, export *model.Export) (bool, error) {
			return uc.enqueueGeneration(ctx, export)
		},
	)
	if taken > 0 {
		slog.InfoContext(ctx, "未投入の queued エクスポートを再投入しました", "count", taken)
	}
	if err != nil {
		return fmt.Errorf("queued エクスポートの回復に失敗: %w", err)
	}
	return nil
}

// recoverStaleStarted converges every export whose attempt has been running for
// longer than the generation timeout plus the grace. An export with attempts
// left goes back to queued and is enqueued again; one that used them up is
// closed as failed, so the screen shows a failure the user can retry from
// instead of a generation that never ends.
//
// [Ja] recoverStaleStarted は、試行が生成の timeout に猶予を足した時間より長く続いて
// いるエクスポートを収束させる。試行が残っているエクスポートは queued へ戻して再投入
// し、使い切ったものは failed として閉じる。これにより画面には、終わらない生成では
// なく、ユーザーが再実行できる失敗が表示される。
func (uc *ReconcileExportsUsecase) recoverStaleStarted(ctx context.Context, now time.Time) error {
	threshold := now.Add(-(GenerateExportTimeout + exportStartedGrace))

	taken, err := takeExportRecoveryCandidates(ctx, uc.limits,
		func(ctx context.Context, cursor *repository.ExportRecoveryCursor, pageSize int32) ([]*model.Export, *repository.ExportRecoveryCursor, error) {
			return uc.exportRepo.ListStaleStarted(ctx, threshold, cursor, pageSize)
		},
		func(ctx context.Context, export *model.Export) (bool, error) {
			if export.AttemptCount >= dispatcher.GenerateExportMaxAttempts {
				return uc.closeStaleStarted(ctx, export)
			}
			return uc.requeueStaleStarted(ctx, export)
		},
	)
	if taken > 0 {
		slog.InfoContext(ctx, "停滞した started エクスポートを回復しました", "count", taken)
	}
	if err != nil {
		return fmt.Errorf("started エクスポートの回復に失敗: %w", err)
	}
	return nil
}

// requeueStaleStarted puts a stale attempt back into queued and enqueues a new
// generation job for it. The transition is guarded by the token the row carried
// when it was read, so a conflicting move made in between leaves the row to
// whoever holds it now.
//
// [Ja] requeueStaleStarted は停滞した試行を queued へ戻し、新しい生成ジョブを投入する。
// 遷移は行を読んだ時点のトークンでガードされるため、その間に競合する遷移が起きていた
// 場合は、現在その行を保持している側に委ねる。
func (uc *ReconcileExportsUsecase) requeueStaleStarted(ctx context.Context, export *model.Export) (bool, error) {
	requeued, err := uc.exportRepo.Requeue(ctx, export.ID, export.UpdatedAt)
	if err != nil {
		return false, fmt.Errorf("停滞したエクスポートの queued への差し戻しに失敗 (export_id: %s): %w", export.ID.String(), err)
	}
	if !requeued {
		return false, nil
	}

	slog.WarnContext(ctx, "停滞したエクスポートを queued へ戻しました",
		"export_id", export.ID.String(),
		"attempt_count", export.AttemptCount,
	)

	// A skipped insert here means the previous attempt's job is still waiting or
	// running, which is exactly the case this row was requeued for. The row is
	// queued now, so the queued stream enqueues it once that job is gone.
	//
	// [Ja] ここで投入が skip されるのは、前の試行のジョブがまだ待機中または実行中の
	// 場合で、この行を queued へ戻したのはまさにその状況である。行は queued に
	// なっているため、そのジョブが消えた後に queued の系統が投入する。
	if _, err := uc.dispatcher.EnqueueGenerateExport(ctx, export.ID.String()); err != nil {
		return false, fmt.Errorf("差し戻したエクスポートの生成ジョブ投入に失敗 (export_id: %s): %w", export.ID.String(), err)
	}
	return true, nil
}

// closeStaleStarted ends an export that used up its attempts. Its object, if an
// attempt uploaded one, is released by this transition and collected by the
// orphan sweep.
//
// [Ja] closeStaleStarted は試行を使い切ったエクスポートを終わらせる。いずれかの試行が
// アップロードしたオブジェクトがあれば、この遷移によって手放され、孤児回収が回収する。
func (uc *ReconcileExportsUsecase) closeStaleStarted(ctx context.Context, export *model.Export) (bool, error) {
	failed, err := uc.exportRepo.MarkFailed(ctx, export.ID, export.UpdatedAt)
	if err != nil {
		return false, fmt.Errorf("試行上限に達したエクスポートの失敗記録に失敗 (export_id: %s): %w", export.ID.String(), err)
	}
	if !failed {
		return false, nil
	}

	slog.WarnContext(ctx, "試行上限に達したエクスポートを失敗として終了しました",
		"export_id", export.ID.String(),
		"attempt_count", export.AttemptCount,
	)
	return true, nil
}

// notifyPendingCompletions enqueues an email job for every notification that
// has remained pending beyond the grace period.
//
// [Ja] notifyPendingCompletions は猶予時間を超えて pending のままの通知について、
// 完了メールジョブを投入する。
func (uc *ReconcileExportsUsecase) notifyPendingCompletions(ctx context.Context, now time.Time) error {
	threshold := now.Add(-exportNotificationGrace)

	taken, err := takeExportRecoveryCandidates(ctx, uc.limits,
		func(ctx context.Context, cursor *repository.ExportCompletionNotificationCursor, pageSize int32) ([]*model.ExportCompletionNotification, *repository.ExportCompletionNotificationCursor, error) {
			return uc.notificationRepo.ListPending(ctx, threshold, cursor, pageSize)
		},
		func(ctx context.Context, notification *model.ExportCompletionNotification) (bool, error) {
			inserted, err := uc.dispatcher.EnqueueSendExportCompletedEmail(ctx, notification.ExportID.String())
			if err != nil {
				return false, fmt.Errorf("完了通知メールジョブの投入に失敗 (export_id: %s): %w", notification.ExportID.String(), err)
			}
			return inserted, nil
		},
	)
	if taken > 0 {
		slog.InfoContext(ctx, "送信待ちのエクスポート完了通知を投入しました", "count", taken)
	}
	if err != nil {
		return fmt.Errorf("送信待ち完了通知の回復に失敗: %w", err)
	}
	return nil
}

// cleanupProfilesWithOldSucceeded enqueues a cleanup job for every profile that
// still holds a succeeded export older than its latest one, recovering the
// cleanups whose insert after a success was lost.
//
// [Ja] cleanupProfilesWithOldSucceeded は、最新のものより古い succeeded の
// エクスポートをまだ持つプロフィールごとに cleanup ジョブを投入し、成功後の投入が
// 失われた掃除を回復する。
func (uc *ReconcileExportsUsecase) cleanupProfilesWithOldSucceeded(ctx context.Context) error {
	taken, err := takeExportRecoveryCandidates(ctx, uc.limits,
		func(ctx context.Context, cursor *model.ProfileID, pageSize int32) ([]model.ProfileID, *model.ProfileID, error) {
			return uc.exportRepo.ListProfileIDsWithOldSucceeded(ctx, cursor, pageSize)
		},
		func(ctx context.Context, profileID model.ProfileID) (bool, error) {
			inserted, err := uc.dispatcher.EnqueueCleanupOldExports(ctx, profileID.String())
			if err != nil {
				return false, fmt.Errorf("旧エクスポート削除ジョブの投入に失敗 (profile_id: %s): %w", profileID.String(), err)
			}
			return inserted, nil
		},
	)
	if taken > 0 {
		slog.InfoContext(ctx, "旧エクスポートを持つプロフィールの削除ジョブを投入しました", "count", taken)
	}
	if err != nil {
		return fmt.Errorf("旧 succeeded エクスポートの回復に失敗: %w", err)
	}
	return nil
}

// enqueueGeneration inserts the unique generation job for an export and reports
// whether this run is the one that inserted it.
//
// [Ja] enqueueGeneration はエクスポートの一意な生成ジョブを投入し、それを投入したのが
// 今回の実行かどうかを返す。
func (uc *ReconcileExportsUsecase) enqueueGeneration(ctx context.Context, export *model.Export) (bool, error) {
	inserted, err := uc.dispatcher.EnqueueGenerateExport(ctx, export.ID.String())
	if err != nil {
		return false, fmt.Errorf("生成ジョブの再投入に失敗 (export_id: %s): %w", export.ID.String(), err)
	}
	return inserted, nil
}

// takeExportRecoveryCandidates walks one recovery stream page by page, hands
// each candidate to take, and stops once the run has taken on as many
// candidates as the budget allows or the walk reaches the end. It returns how
// many candidates were taken on.
//
// take reports whether the candidate became this run's work: a candidate whose
// unique job already exists, or whose guarded transition lost to a concurrent
// one, is not counted and does not spend budget. The cursor keeps advancing
// past those candidates, so the ones behind a stuck head are still reached
// within the same run.
//
// [Ja] takeExportRecoveryCandidates は 1 系統の回復候補をページ単位で走査し、各候補を
// take へ渡す。予算の分だけ候補を引き受けるか、走査が終端に達したところで止まる。
// 戻り値は引き受けた候補数。
//
// take は、その候補が今回の実行の処理になったかどうかを返す。一意ジョブがすでに存在
// する候補や、ガード付き遷移が並行処理に負けた候補は数えず、予算も使わない。cursor は
// そうした候補を越えて進み続けるため、停滞した先頭候補の後ろにある候補にも同じ実行の
// 中で到達できる。
func takeExportRecoveryCandidates[Candidate any, Cursor any](
	ctx context.Context,
	limits ExportRecoveryLimits,
	list func(ctx context.Context, cursor *Cursor, pageSize int32) ([]Candidate, *Cursor, error),
	take func(ctx context.Context, candidate Candidate) (bool, error),
) (int, error) {
	var cursor *Cursor
	taken := 0

	for taken < limits.Budget {
		candidates, next, err := list(ctx, cursor, limits.PageSize)
		if err != nil {
			return taken, err
		}

		for _, candidate := range candidates {
			ok, err := take(ctx, candidate)
			if err != nil {
				return taken, err
			}
			if !ok {
				continue
			}
			taken++
			if taken >= limits.Budget {
				break
			}
		}

		if next == nil {
			break
		}
		cursor = next
	}

	return taken, nil
}
