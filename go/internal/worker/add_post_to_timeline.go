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

// AddPostToTimelineWorkerはホームタイムラインへの追加ワーカー
type AddPostToTimelineWorker struct {
	river.WorkerDefaults[dispatcher.AddPostToTimelineArgs]
	uc *usecase.AddPostToTimelineUsecase
}

// NewAddPostToTimelineWorkerは新しいAddPostToTimelineWorkerを作成する
func NewAddPostToTimelineWorker(uc *usecase.AddPostToTimelineUsecase) *AddPostToTimelineWorker {
	return &AddPostToTimelineWorker{uc: uc}
}

// Workはジョブ引数を変換し、タイムライン追加のUseCaseを実行する。
func (w *AddPostToTimelineWorker) Work(ctx context.Context, job *river.Job[dispatcher.AddPostToTimelineArgs]) error {
	profileID, err := uuid.Parse(job.Args.ProfileID)
	if err != nil {
		return fmt.Errorf("profile_idのパースに失敗: %w", err)
	}
	postID, err := uuid.Parse(job.Args.PostID)
	if err != nil {
		return fmt.Errorf("post_idのパースに失敗: %w", err)
	}

	return w.uc.Execute(ctx, usecase.AddPostToTimelineInput{
		ProfileID: model.ProfileID(profileID),
		PostID:    model.PostID(postID),
	})
}
