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

// SendExportCompletedEmailExecutor delivers one export's pending completion
// notification.
//
// [Ja] SendExportCompletedEmailExecutor は 1 件のエクスポートの送信待ち完了通知を
// 配信する。
type SendExportCompletedEmailExecutor interface {
	Execute(context.Context, model.ExportID) error
}

// SendExportCompletedEmailWorker runs the delivery of an export's completion
// notification.
//
// [Ja] SendExportCompletedEmailWorker はエクスポートの完了通知の配信を実行する。
type SendExportCompletedEmailWorker struct {
	river.WorkerDefaults[dispatcher.SendExportCompletedEmailArgs]
	uc SendExportCompletedEmailExecutor
}

// NewSendExportCompletedEmailWorker creates a SendExportCompletedEmailWorker.
//
// [Ja] NewSendExportCompletedEmailWorker は SendExportCompletedEmailWorker を
// 生成する。
func NewSendExportCompletedEmailWorker(uc SendExportCompletedEmailExecutor) *SendExportCompletedEmailWorker {
	return &SendExportCompletedEmailWorker{uc: uc}
}

// Timeout bounds one delivery attempt so that a mail provider that stopped
// answering releases the default-queue worker it holds.
//
// [Ja] Timeout は配信 1 回の試行を区切り、応答しなくなったメールプロバイダーが占有する
// 既定キューの worker を解放させる。
func (w *SendExportCompletedEmailWorker) Timeout(*river.Job[dispatcher.SendExportCompletedEmailArgs]) time.Duration {
	return usecase.SendExportCompletedEmailTimeout
}

// Work delivers the completion notification of the export the job names.
// Whether that notification is still pending is decided on every run from the
// outbox, so the argument only carries the export.
//
// [Ja] Work はジョブが示すエクスポートの完了通知を配信する。その通知がまだ送信待ちかは
// 毎回の実行で outbox から決めるため、引数が運ぶのはエクスポートだけである。
func (w *SendExportCompletedEmailWorker) Work(ctx context.Context, job *river.Job[dispatcher.SendExportCompletedEmailArgs]) error {
	exportID, err := uuid.Parse(job.Args.ExportID)
	if err != nil {
		return fmt.Errorf("export_id のパースに失敗: %w", err)
	}

	return w.uc.Execute(ctx, model.ExportID(exportID))
}
