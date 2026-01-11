// Package email はメール送信機能を提供します
package email

import (
	"bytes"
	"context"
	"fmt"

	"github.com/a-h/templ"
	"github.com/resend/resend-go/v2"
)

// Sender はメール送信を行うインターフェース
type Sender interface {
	// SendEmailConfirmation は確認コードを含むメールを送信する
	SendEmailConfirmation(ctx context.Context, input SendEmailConfirmationInput) error
}

// SendEmailConfirmationInput は確認メール送信の入力
type SendEmailConfirmationInput struct {
	To       string          // 送信先メールアドレス
	Subject  string          // 件名
	HTMLBody templ.Component // メール本文（HTML形式）
	TextBody templ.Component // メール本文（テキスト形式）
}

// ResendSender はResend APIを使用してメールを送信する
type ResendSender struct {
	client *resend.Client
	from   string
}

// NewResendSender は新しいResendSenderを作成する
func NewResendSender(apiKey, from string) *ResendSender {
	return &ResendSender{
		client: resend.NewClient(apiKey),
		from:   from,
	}
}

// SendEmailConfirmation は確認コードを含むメールを送信する
func (s *ResendSender) SendEmailConfirmation(ctx context.Context, input SendEmailConfirmationInput) error {
	// HTMLテンプレートをレンダリング
	var htmlBuf bytes.Buffer
	if err := input.HTMLBody.Render(ctx, &htmlBuf); err != nil {
		return fmt.Errorf("HTMLテンプレートのレンダリングに失敗しました: %w", err)
	}

	// テキストテンプレートをレンダリング
	var textBuf bytes.Buffer
	if err := input.TextBody.Render(ctx, &textBuf); err != nil {
		return fmt.Errorf("テキストテンプレートのレンダリングに失敗しました: %w", err)
	}

	// Resend APIを使用してメール送信
	params := &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{input.To},
		Subject: input.Subject,
		Html:    htmlBuf.String(),
		Text:    textBuf.String(),
	}

	_, err := s.client.Emails.SendWithContext(ctx, params)
	if err != nil {
		return fmt.Errorf("メール送信に失敗しました: %w", err)
	}

	return nil
}

// NoopSender はメールを送信しないダミー実装（テスト用）
type NoopSender struct {
	// SentEmails は送信されたメールを記録する（テスト用）
	SentEmails []SendEmailConfirmationInput
}

// NewNoopSender は新しいNoopSenderを作成する
func NewNoopSender() *NoopSender {
	return &NoopSender{
		SentEmails: make([]SendEmailConfirmationInput, 0),
	}
}

// SendEmailConfirmation はメールを送信せず、記録のみ行う
func (s *NoopSender) SendEmailConfirmation(_ context.Context, input SendEmailConfirmationInput) error {
	s.SentEmails = append(s.SentEmails, input)
	return nil
}
