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

// SendExportCompletedEmailExecutorは1件のエクスポートの送信待ち完了通知を
// 配信する。
type SendExportCompletedEmailExecutor interface {
	Execute(context.Context, model.ExportID) error
}

// SendExportCompletedEmailWorkerはエクスポートの完了通知の配信を実行する。
type SendExportCompletedEmailWorker struct {
	river.WorkerDefaults[dispatcher.SendExportCompletedEmailArgs]
	uc SendExportCompletedEmailExecutor
}

// NewSendExportCompletedEmailWorkerはSendExportCompletedEmailWorkerを
// 生成する。
func NewSendExportCompletedEmailWorker(uc SendExportCompletedEmailExecutor) *SendExportCompletedEmailWorker {
	return &SendExportCompletedEmailWorker{uc: uc}
}

// Timeoutは配信1回の試行を区切り、応答しなくなったメールプロバイダーが占有する
// 既定キューのworkerを解放させる。
func (w *SendExportCompletedEmailWorker) Timeout(*river.Job[dispatcher.SendExportCompletedEmailArgs]) time.Duration {
	return usecase.SendExportCompletedEmailTimeout
}

// Workはジョブが示すエクスポートの完了通知を配信する。その通知がまだ送信待ちかは
// 毎回の実行でoutboxから決めるため、引数が運ぶのはエクスポートだけである。
func (w *SendExportCompletedEmailWorker) Work(ctx context.Context, job *river.Job[dispatcher.SendExportCompletedEmailArgs]) error {
	exportID, err := uuid.Parse(job.Args.ExportID)
	if err != nil {
		return fmt.Errorf("export_idのパースに失敗: %w", err)
	}

	return w.uc.Execute(ctx, model.ExportID(exportID))
}
