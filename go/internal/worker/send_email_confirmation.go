package worker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/riverqueue/river"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// SendEmailConfirmationWorker はメール確認コード送信を処理するワーカー
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

// Work はメール確認コード送信ジョブを実行する
func (w *SendEmailConfirmationWorker) Work(ctx context.Context, job *river.Job[dispatcher.SendEmailConfirmationArgs]) error {
	args := job.Args

	slog.InfoContext(ctx, "メール確認コード送信ジョブを開始します",
		"email", args.Email,
		"locale", args.Locale,
	)

	if err := w.uc.Execute(ctx, usecase.SendEmailConfirmationInput{
		Email:  args.Email,
		Code:   args.Code,
		Locale: args.Locale,
	}); err != nil {
		return fmt.Errorf("メール確認コード送信ジョブの実行に失敗: %w", err)
	}

	return nil
}
