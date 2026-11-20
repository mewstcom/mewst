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

// GenerateExportExecutorは1件のエクスポートのアーカイブを生成する。workerが
// UseCaseそのものではなくこのinterfaceに依存するのは、ジョブから入力への変換を、
// 生成が必要とするDBとオブジェクトストレージ抜きでテストできるようにするため。
type GenerateExportExecutor interface {
	Execute(ctx context.Context, input usecase.GenerateExportInput) error
}

// GenerateExportWorkerはエクスポート生成ジョブを実行する。
type GenerateExportWorker struct {
	river.WorkerDefaults[dispatcher.GenerateExportArgs]
	uc GenerateExportExecutor
}

// NewGenerateExportWorkerはGenerateExportWorkerを作成する。
func NewGenerateExportWorker(uc GenerateExportExecutor) *GenerateExportWorker {
	return &GenerateExportWorker{uc: uc}
}

// Timeoutは生成1回の試行を区切り、停止した試行が、プロセスが生きている限り
// 他プロフィールのエクスポートを塞ぎ続けるのではなく、workerが1つだけのexport
// キューを解放するようにする。
func (w *GenerateExportWorker) Timeout(*river.Job[dispatcher.GenerateExportArgs]) time.Duration {
	return usecase.GenerateExportTimeout
}

// Workはジョブ引数を変換し、エクスポート生成のUseCaseを実行する。失敗で
// エクスポートを終わらせるかどうかはRiverの試行回数が決める。UseCaseがfailed
// として閉じるのは最終試行のときだけで、それ以前は次の試行のために行をstartedの
// まま残す。
func (w *GenerateExportWorker) Work(ctx context.Context, job *river.Job[dispatcher.GenerateExportArgs]) error {
	exportID, err := uuid.Parse(job.Args.ExportID)
	if err != nil {
		return fmt.Errorf("export_idのパースに失敗: %w", err)
	}

	return w.uc.Execute(ctx, usecase.GenerateExportInput{
		ExportID:       model.ExportID(exportID),
		IsFinalAttempt: job.Attempt >= job.MaxAttempts,
	})
}
