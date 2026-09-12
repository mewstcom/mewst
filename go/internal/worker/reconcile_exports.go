package worker

import (
	"context"
	"time"

	"github.com/riverqueue/river"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// ReconcileExportsWorker runs the periodic export reconciliation.
//
// [Ja] ReconcileExportsWorker はエクスポートの定期リコンシリエーションを実行する。
type ReconcileExportsWorker struct {
	river.WorkerDefaults[dispatcher.ReconcileExportsArgs]
	uc *usecase.ReconcileExportsUsecase
}

// NewReconcileExportsWorker creates a ReconcileExportsWorker.
//
// [Ja] NewReconcileExportsWorker は ReconcileExportsWorker を生成する。
func NewReconcileExportsWorker(uc *usecase.ReconcileExportsUsecase) *ReconcileExportsWorker {
	return &ReconcileExportsWorker{uc: uc}
}

// Timeout bounds one reconciliation run so that a run which stopped making
// progress releases its worker before the next scheduled run.
//
// [Ja] Timeout はリコンシリエーション 1 回の実行を区切り、前進しなくなった実行が
// 次回の定期実行より前に worker を解放するようにする。
func (w *ReconcileExportsWorker) Timeout(*river.Job[dispatcher.ReconcileExportsArgs]) time.Duration {
	return usecase.ReconcileExportsTimeout
}

// Work runs the reconciliation. The job carries no arguments: the outstanding
// work is derived from the exports table on every run.
//
// [Ja] Work はリコンシリエーションを実行する。ジョブは引数を持たず、未処理の作業は
// 毎回の実行で exports テーブルから導出される。
func (w *ReconcileExportsWorker) Work(ctx context.Context, _ *river.Job[dispatcher.ReconcileExportsArgs]) error {
	return w.uc.Execute(ctx)
}
