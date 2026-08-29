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

// GenerateExportExecutor generates one export's archive. The worker depends on
// this interface rather than on the use case itself so that the conversion from
// a job to its input can be tested without the database and object storage the
// generation needs.
//
// [Ja] GenerateExportExecutor は 1 件のエクスポートのアーカイブを生成する。worker が
// UseCase そのものではなくこの interface に依存するのは、ジョブから入力への変換を、
// 生成が必要とする DB とオブジェクトストレージ抜きでテストできるようにするため。
type GenerateExportExecutor interface {
	Execute(ctx context.Context, input usecase.GenerateExportInput) error
}

// GenerateExportWorker runs export generation jobs.
//
// [Ja] GenerateExportWorker はエクスポート生成ジョブを実行する。
type GenerateExportWorker struct {
	river.WorkerDefaults[dispatcher.GenerateExportArgs]
	uc GenerateExportExecutor
}

// NewGenerateExportWorker creates a GenerateExportWorker.
//
// [Ja] NewGenerateExportWorker は GenerateExportWorker を作成する。
func NewGenerateExportWorker(uc GenerateExportExecutor) *GenerateExportWorker {
	return &GenerateExportWorker{uc: uc}
}

// Timeout bounds one generation attempt so that a stalled one releases the
// single-worker export queue instead of blocking every other profile's export
// for as long as the process lives.
//
// [Ja] Timeout は生成 1 回の試行を区切り、停止した試行が、プロセスが生きている限り
// 他プロフィールのエクスポートを塞ぎ続けるのではなく、worker が 1 つだけの export
// キューを解放するようにする。
func (w *GenerateExportWorker) Timeout(*river.Job[dispatcher.GenerateExportArgs]) time.Duration {
	return usecase.GenerateExportTimeout
}

// Work converts the job argument and runs the export generation UseCase. The
// attempt River is on decides whether a failure ends the export: only on the
// last one does the UseCase close it as failed, and before that the row stays
// started for the next attempt.
//
// [Ja] Work はジョブ引数を変換し、エクスポート生成の UseCase を実行する。失敗で
// エクスポートを終わらせるかどうかは River の試行回数が決める。UseCase が failed
// として閉じるのは最終試行のときだけで、それ以前は次の試行のために行を started の
// まま残す。
func (w *GenerateExportWorker) Work(ctx context.Context, job *river.Job[dispatcher.GenerateExportArgs]) error {
	exportID, err := uuid.Parse(job.Args.ExportID)
	if err != nil {
		return fmt.Errorf("export_id のパースに失敗: %w", err)
	}

	return w.uc.Execute(ctx, usecase.GenerateExportInput{
		ExportID:       model.ExportID(exportID),
		IsFinalAttempt: job.Attempt >= job.MaxAttempts,
	})
}
