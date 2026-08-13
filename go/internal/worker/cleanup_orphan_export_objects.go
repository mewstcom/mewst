package worker

import (
	"context"
	"time"

	"github.com/riverqueue/river"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// CleanupOrphanExportObjectsWorker runs the periodic sweep that deletes export
// objects no export retains any more.
//
// [Ja] CleanupOrphanExportObjectsWorker は、どのエクスポートからも保持されなくなった
// エクスポートオブジェクトを削除する定期的な掃除を実行する。
type CleanupOrphanExportObjectsWorker struct {
	river.WorkerDefaults[dispatcher.CleanupOrphanExportObjectsArgs]
	uc *usecase.CleanupOrphanExportObjectsUsecase
}

// NewCleanupOrphanExportObjectsWorker creates a
// CleanupOrphanExportObjectsWorker.
//
// [Ja] NewCleanupOrphanExportObjectsWorker は
// CleanupOrphanExportObjectsWorker を生成する。
func NewCleanupOrphanExportObjectsWorker(uc *usecase.CleanupOrphanExportObjectsUsecase) *CleanupOrphanExportObjectsWorker {
	return &CleanupOrphanExportObjectsWorker{uc: uc}
}

// Timeout bounds one sweep, whose cost follows the number of stored archives
// rather than a fixed amount of work.
//
// [Ja] Timeout は掃除 1 回の実行を区切る。掃除のコストは固定量ではなく保存済み
// アーカイブ数に従うため。
func (w *CleanupOrphanExportObjectsWorker) Timeout(*river.Job[dispatcher.CleanupOrphanExportObjectsArgs]) time.Duration {
	return usecase.CleanupOrphanExportObjectsTimeout
}

// Work runs the sweep from where the job says to resume. Which of the listed
// objects are orphans is decided on every run by matching the object storage
// against the exports table; the argument only carries the position.
//
// [Ja] Work はジョブが示す再開位置から掃除を実行する。一覧されたどのオブジェクトが
// 孤児かは毎回の実行でオブジェクトストレージと exports テーブルを照合して求め、引数が
// 運ぶのは位置だけである。
func (w *CleanupOrphanExportObjectsWorker) Work(ctx context.Context, job *river.Job[dispatcher.CleanupOrphanExportObjectsArgs]) error {
	return w.uc.Execute(ctx, job.Args.StartAfter)
}
