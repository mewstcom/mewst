package worker

import (
	"context"
	"time"

	"github.com/riverqueue/river"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// CleanupOrphanExportObjectsWorkerは、どのエクスポートからも保持されなくなった
// エクスポートオブジェクトを削除する定期的な掃除を実行する。
type CleanupOrphanExportObjectsWorker struct {
	river.WorkerDefaults[dispatcher.CleanupOrphanExportObjectsArgs]
	uc *usecase.CleanupOrphanExportObjectsUsecase
}

// NewCleanupOrphanExportObjectsWorkerは
// CleanupOrphanExportObjectsWorkerを生成する。
func NewCleanupOrphanExportObjectsWorker(uc *usecase.CleanupOrphanExportObjectsUsecase) *CleanupOrphanExportObjectsWorker {
	return &CleanupOrphanExportObjectsWorker{uc: uc}
}

// Timeoutは掃除1回の実行を区切る。掃除のコストは固定量ではなく保存済み
// アーカイブ数に従うため。
func (w *CleanupOrphanExportObjectsWorker) Timeout(*river.Job[dispatcher.CleanupOrphanExportObjectsArgs]) time.Duration {
	return usecase.CleanupOrphanExportObjectsTimeout
}

// Workはジョブが示す再開位置から掃除を実行する。一覧されたどのオブジェクトが
// 孤児かは毎回の実行でオブジェクトストレージとexportsテーブルを照合して求め、引数が
// 運ぶのは位置だけである。
func (w *CleanupOrphanExportObjectsWorker) Work(ctx context.Context, job *river.Job[dispatcher.CleanupOrphanExportObjectsArgs]) error {
	return w.uc.Execute(ctx, job.Args.StartAfter)
}
