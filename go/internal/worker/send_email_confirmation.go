package worker

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"

	"github.com/riverqueue/river"

	"github.com/mewstcom/mewst/go/internal/email"
	"github.com/mewstcom/mewst/go/internal/templates/emails/email_confirmation"
)

// SendEmailConfirmationArgs はメール確認コード送信ジョブの引数
type SendEmailConfirmationArgs struct {
	Email  string `json:"email"`
	Code   string `json:"code"`
	Locale string `json:"locale"`
}

// Kind はジョブの種類を返す（riverのインターフェース実装）
func (SendEmailConfirmationArgs) Kind() string {
	return "send_email_confirmation"
}

// InsertOpts はジョブのオプションを返す（riverのインターフェース実装）
func (SendEmailConfirmationArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       river.QueueDefault,
		MaxAttempts: 5,
	}
}

// SendEmailConfirmationWorker はメール確認コード送信を処理するワーカー
type SendEmailConfirmationWorker struct {
	river.WorkerDefaults[SendEmailConfirmationArgs]
	sender email.Sender
}

// NewSendEmailConfirmationWorker は新しいSendEmailConfirmationWorkerを作成する
func NewSendEmailConfirmationWorker(sender email.Sender) *SendEmailConfirmationWorker {
	return &SendEmailConfirmationWorker{
		sender: sender,
	}
}

// Work はメール確認コード送信ジョブを実行する
func (w *SendEmailConfirmationWorker) Work(ctx context.Context, job *river.Job[SendEmailConfirmationArgs]) error {
	args := job.Args

	slog.InfoContext(ctx, "メール確認コード送信ジョブを開始します",
		"email", args.Email,
		"locale", args.Locale,
	)

	// メールアドレスの検証
	if args.Email == "" {
		slog.ErrorContext(ctx, "メールアドレスが空です")
		return fmt.Errorf("メールアドレスが空です")
	}

	// テンプレートをレンダリング
	htmlBody, textBody, err := w.renderEmailTemplates(ctx, args)
	if err != nil {
		slog.ErrorContext(ctx, "メールテンプレートのレンダリングに失敗しました",
			"email", args.Email,
			"error", err,
		)
		return fmt.Errorf("メールテンプレートのレンダリングに失敗: %w", err)
	}

	// メール件名をロケールに基づいて選択
	subject := getEmailSubject(args.Locale)

	// メール送信
	err = w.sender.SendRaw(ctx, email.SendRawInput{
		To:       args.Email,
		Subject:  subject,
		HTMLBody: htmlBody,
		TextBody: textBody,
	})
	if err != nil {
		slog.ErrorContext(ctx, "メール送信に失敗しました",
			"email", args.Email,
			"error", err,
		)
		return fmt.Errorf("メール送信に失敗: %w", err)
	}

	slog.InfoContext(ctx, "メール確認コードを送信しました",
		"email", args.Email,
	)

	return nil
}

// renderEmailTemplates はロケールに基づいてメールテンプレートをレンダリングする
func (w *SendEmailConfirmationWorker) renderEmailTemplates(ctx context.Context, args SendEmailConfirmationArgs) (htmlBody, textBody string, err error) {
	var htmlBuf bytes.Buffer
	var textBuf bytes.Buffer

	switch args.Locale {
	case "en":
		if err := email_confirmation.EnHTML(args.Email, args.Code).Render(ctx, &htmlBuf); err != nil {
			return "", "", fmt.Errorf("HTMLテンプレートのレンダリングに失敗しました: %w", err)
		}
		if err := email_confirmation.EnText(args.Email, args.Code).Render(ctx, &textBuf); err != nil {
			return "", "", fmt.Errorf("テキストテンプレートのレンダリングに失敗しました: %w", err)
		}
	default:
		if err := email_confirmation.JaHTML(args.Email, args.Code).Render(ctx, &htmlBuf); err != nil {
			return "", "", fmt.Errorf("HTMLテンプレートのレンダリングに失敗しました: %w", err)
		}
		if err := email_confirmation.JaText(args.Email, args.Code).Render(ctx, &textBuf); err != nil {
			return "", "", fmt.Errorf("テキストテンプレートのレンダリングに失敗しました: %w", err)
		}
	}

	return htmlBuf.String(), textBuf.String(), nil
}

// getEmailSubject はロケールに基づいてメール件名を返す
func getEmailSubject(locale string) string {
	switch locale {
	case "en":
		return "[Mewst] Confirmation code"
	default:
		return "[Mewst] 確認用コード"
	}
}
