package worker

import (
	"context"
	"time"

	"github.com/riverqueue/river"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// ReconcileExportsWorkerはエクスポートの定期リコンシリエーションを実行する。
type ReconcileExportsWorker struct {
	river.WorkerDefaults[dispatcher.ReconcileExportsArgs]
	uc *usecase.ReconcileExportsUsecase
}

// NewReconcileExportsWorkerはReconcileExportsWorkerを生成する。
func NewReconcileExportsWorker(uc *usecase.ReconcileExportsUsecase) *ReconcileExportsWorker {
	return &ReconcileExportsWorker{uc: uc}
}

// Timeoutはリコンシリエーション1回の実行を区切り、前進しなくなった実行が
// 次回の定期実行より前にworkerを解放するようにする。
func (w *ReconcileExportsWorker) Timeout(*river.Job[dispatcher.ReconcileExportsArgs]) time.Duration {
	return usecase.ReconcileExportsTimeout
}

// Workはリコンシリエーションを実行する。ジョブは引数を持たず、未処理の作業は
// 毎回の実行でexportsテーブルから導出される。
func (w *ReconcileExportsWorker) Work(ctx context.Context, _ *river.Job[dispatcher.ReconcileExportsArgs]) error {
	return w.uc.Execute(ctx)
}
