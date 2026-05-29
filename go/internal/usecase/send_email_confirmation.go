package usecase

import (
	"context"
	"fmt"
	"log/slog"
)

// EmailConfirmationSender はメール確認コードの送信を抽象化するインターフェース
// emailパッケージのConfirmationSenderが実装する
type EmailConfirmationSender interface {
	Send(ctx context.Context, to, code, locale string) error
}

// SendEmailConfirmationUsecase はメール確認コードのメール送信ユースケース
type SendEmailConfirmationUsecase struct {
	sender EmailConfirmationSender
}

// NewSendEmailConfirmationUsecase は SendEmailConfirmationUsecase を生成する
func NewSendEmailConfirmationUsecase(sender EmailConfirmationSender) *SendEmailConfirmationUsecase {
	return &SendEmailConfirmationUsecase{
		sender: sender,
	}
}

// SendEmailConfirmationInput はメール確認コード送信の入力パラメータ
type SendEmailConfirmationInput struct {
	Email  string
	Code   string
	Locale string
}

// Execute はメール確認コードのメールを送信する
func (uc *SendEmailConfirmationUsecase) Execute(ctx context.Context, input SendEmailConfirmationInput) error {
	if err := uc.sender.Send(ctx, input.Email, input.Code, input.Locale); err != nil {
		slog.ErrorContext(ctx, "メール確認コード送信ユースケースの実行に失敗", "error", err, "email", input.Email)
		return fmt.Errorf("メール確認コード送信ユースケースの実行に失敗: %w", err)
	}

	slog.InfoContext(ctx, "メール確認コードを送信しました", "email", input.Email)

	return nil
}
