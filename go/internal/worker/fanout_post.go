package worker

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// FanoutPostWorkerはタイムライン配信ワーカー
type FanoutPostWorker struct {
	river.WorkerDefaults[dispatcher.FanoutPostArgs]
	uc *usecase.FanoutPostUsecase
}

// NewFanoutPostWorkerは新しいFanoutPostWorkerを作成する
func NewFanoutPostWorker(uc *usecase.FanoutPostUsecase) *FanoutPostWorker {
	return &FanoutPostWorker{uc: uc}
}

// Workはジョブ引数を変換し、fanoutのUseCaseを実行する。
func (w *FanoutPostWorker) Work(ctx context.Context, job *river.Job[dispatcher.FanoutPostArgs]) error {
	postID, err := uuid.Parse(job.Args.PostID)
	if err != nil {
		return fmt.Errorf("post_idのパースに失敗: %w", err)
	}

	return w.uc.Execute(ctx, usecase.FanoutPostInput{
		PostID: model.PostID(postID),
	})
}
