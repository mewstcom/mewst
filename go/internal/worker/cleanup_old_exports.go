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

// CleanupOldExportsExecutor removes superseded exports for one profile.
//
// [Ja] CleanupOldExportsExecutor は 1 プロフィールの置き換え済み export を削除する。
type CleanupOldExportsExecutor interface {
	Execute(context.Context, model.ProfileID) error
}

// CleanupOldExportsWorker runs the deletion of a profile's superseded succeeded
// exports.
//
// [Ja] CleanupOldExportsWorker は、プロフィールの置き換え済み succeeded export の
// 削除を実行する。
type CleanupOldExportsWorker struct {
	river.WorkerDefaults[dispatcher.CleanupOldExportsArgs]
	uc CleanupOldExportsExecutor
}

// NewCleanupOldExportsWorker creates a CleanupOldExportsWorker.
//
// [Ja] NewCleanupOldExportsWorker は CleanupOldExportsWorker を生成する。
func NewCleanupOldExportsWorker(uc CleanupOldExportsExecutor) *CleanupOldExportsWorker {
	return &CleanupOldExportsWorker{uc: uc}
}

// Timeout bounds one cleanup run so that a storage that stopped answering
// releases the default-queue worker it holds.
//
// [Ja] Timeout は掃除 1 回の実行を区切り、応答しなくなったストレージが占有する
// 既定キューの worker を解放させる。
func (w *CleanupOldExportsWorker) Timeout(*river.Job[dispatcher.CleanupOldExportsArgs]) time.Duration {
	return usecase.CleanupOldExportsTimeout
}

// Work runs the cleanup for the profile the job names. Which of the profile's
// exports are deleted is decided on every run from the exports table, so the
// argument only carries the profile.
//
// [Ja] Work はジョブが示すプロフィールの掃除を実行する。そのプロフィールのどの
// エクスポートを削除するかは毎回の実行で exports テーブルから決めるため、引数が
// 運ぶのはプロフィールだけである。
func (w *CleanupOldExportsWorker) Work(ctx context.Context, job *river.Job[dispatcher.CleanupOldExportsArgs]) error {
	profileID, err := uuid.Parse(job.Args.ProfileID)
	if err != nil {
		return fmt.Errorf("profile_id のパースに失敗: %w", err)
	}

	return w.uc.Execute(ctx, model.ProfileID(profileID))
}
