package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// CleanupOldExportsExecutorは1プロフィールの置き換え済みexportを削除する。
type CleanupOldExportsExecutor interface {
	Execute(context.Context, model.ProfileID) error
}

// CleanupOldExportsWorkerは、プロフィールの置き換え済みsucceeded exportの
// 削除を実行する。
type CleanupOldExportsWorker struct {
	river.WorkerDefaults[dispatcher.CleanupOldExportsArgs]
	uc CleanupOldExportsExecutor
}

// NewCleanupOldExportsWorkerはCleanupOldExportsWorkerを生成する。
func NewCleanupOldExportsWorker(uc CleanupOldExportsExecutor) *CleanupOldExportsWorker {
	return &CleanupOldExportsWorker{uc: uc}
}

// Timeoutは掃除1回の実行を区切り、応答しなくなったストレージが占有する
// 既定キューのworkerを解放させる。
func (w *CleanupOldExportsWorker) Timeout(*river.Job[dispatcher.CleanupOldExportsArgs]) time.Duration {
	return usecase.CleanupOldExportsTimeout
}

// Workはジョブが示すプロフィールの掃除を実行する。そのプロフィールのどの
// エクスポートを削除するかは毎回の実行でexportsテーブルから決めるため、引数が
// 運ぶのはプロフィールだけである。
func (w *CleanupOldExportsWorker) Work(ctx context.Context, job *river.Job[dispatcher.CleanupOldExportsArgs]) error {
	profileID, err := uuid.Parse(job.Args.ProfileID)
	if err != nil {
		return fmt.Errorf("profile_idのパースに失敗: %w", err)
	}

	return w.uc.Execute(ctx, model.ProfileID(profileID))
}
