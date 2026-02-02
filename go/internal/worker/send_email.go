package worker

import (
	"context"

	"github.com/riverqueue/river"

	"github.com/mewstcom/mewst/internal/email"
)

// SendEmailArgs はメール送信ジョブの引数
type SendEmailArgs struct {
	To       string `json:"to"`
	Subject  string `json:"subject"`
	HTMLBody string `json:"html_body"`
	TextBody string `json:"text_body"`
}

// Kind はジョブの種類を返す（riverのインターフェース実装）
func (SendEmailArgs) Kind() string {
	return "send_email"
}

// InsertOpts はジョブのオプションを返す（riverのインターフェース実装）
func (SendEmailArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       river.QueueDefault,
		MaxAttempts: 5, // 最大5回リトライ
	}
}

// SendEmailWorker はメール送信を処理するワーカー
type SendEmailWorker struct {
	river.WorkerDefaults[SendEmailArgs]
	sender email.Sender
}

// NewSendEmailWorker は新しいSendEmailWorkerを作成する
func NewSendEmailWorker(sender email.Sender) *SendEmailWorker {
	return &SendEmailWorker{
		sender: sender,
	}
}

// Work はメール送信ジョブを実行する
func (w *SendEmailWorker) Work(ctx context.Context, job *river.Job[SendEmailArgs]) error {
	return w.sender.SendRaw(ctx, email.SendRawInput{
		To:       job.Args.To,
		Subject:  job.Args.Subject,
		HTMLBody: job.Args.HTMLBody,
		TextBody: job.Args.TextBody,
	})
}
