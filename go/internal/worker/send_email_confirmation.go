package worker

import (
	"context"

	"github.com/riverqueue/river"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// SendEmailConfirmationWorker はメール確認コード送信ワーカー
type SendEmailConfirmationWorker struct {
	river.WorkerDefaults[dispatcher.SendEmailConfirmationArgs]
	uc *usecase.SendEmailConfirmationUsecase
}

// NewSendEmailConfirmationWorker は新しいSendEmailConfirmationWorkerを作成する
func NewSendEmailConfirmationWorker(uc *usecase.SendEmailConfirmationUsecase) *SendEmailConfirmationWorker {
	return &SendEmailConfirmationWorker{
		uc: uc,
	}
}

// Work はメール確認コードを送信する
func (w *SendEmailConfirmationWorker) Work(ctx context.Context, job *river.Job[dispatcher.SendEmailConfirmationArgs]) error {
	return w.uc.Execute(ctx, usecase.SendEmailConfirmationInput{
		Email:  job.Args.Email,
		Code:   job.Args.Code,
		Locale: job.Args.Locale,
	})
}
