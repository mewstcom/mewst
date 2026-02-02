// Package email はメール送信機能を提供します
package email

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/a-h/templ"
	"github.com/resend/resend-go/v2"
)

// Sender はメール送信を行うインターフェース
type Sender interface {
	// Send はメールを送信する（templ.Componentを使用）
	Send(ctx context.Context, input SendInput) error
	// SendRaw はレンダリング済みの文字列でメールを送信する（Worker用）
	SendRaw(ctx context.Context, input SendRawInput) error
}

// SendInput はメール送信の入力（templ.Componentを使用）
type SendInput struct {
	To       string          // 送信先メールアドレス
	Subject  string          // 件名
	HTMLBody templ.Component // メール本文（HTML形式）
	TextBody templ.Component // メール本文（テキスト形式）
}

// SendRawInput はレンダリング済み文字列でのメール送信の入力（Worker用）
type SendRawInput struct {
	To       string // 送信先メールアドレス
	Subject  string // 件名
	HTMLBody string // メール本文（HTML形式、レンダリング済み）
	TextBody string // メール本文（テキスト形式、レンダリング済み）
}

// ResendSender はResend APIを使用してメールを送信する
type ResendSender struct {
	client    *resend.Client
	fromEmail string
	fromName  string
}

// NewResendSender は新しいResendSenderを作成する
func NewResendSender(apiKey, fromEmail, fromName string) *ResendSender {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}
	return &ResendSender{
		client:    resend.NewCustomClient(httpClient, apiKey),
		fromEmail: fromEmail,
		fromName:  fromName,
	}
}

// from はFromアドレスを生成する
// fromNameが設定されている場合は「名前 <メール>」形式、そうでない場合はメールアドレスのみ
func (s *ResendSender) from() string {
	if s.fromName != "" {
		return fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail)
	}
	return s.fromEmail
}

// Send はメールを送信する（templ.Componentを使用）
func (s *ResendSender) Send(ctx context.Context, input SendInput) error {
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

	// SendRawを呼び出して実際に送信
	return s.SendRaw(ctx, SendRawInput{
		To:       input.To,
		Subject:  input.Subject,
		HTMLBody: htmlBuf.String(),
		TextBody: textBuf.String(),
	})
}

// SendRaw はレンダリング済みの文字列でメールを送信する（Worker用）
func (s *ResendSender) SendRaw(ctx context.Context, input SendRawInput) error {
	params := &resend.SendEmailRequest{
		From:    s.from(),
		To:      []string{input.To},
		Subject: input.Subject,
		Html:    input.HTMLBody,
		Text:    input.TextBody,
	}

	_, err := s.client.Emails.SendWithContext(ctx, params)
	if err != nil {
		return fmt.Errorf("メール送信に失敗しました: %w", err)
	}

	return nil
}

// NoopSender はメールを送信しないダミー実装（テスト用）
type NoopSender struct {
	// SentEmails は送信されたメールを記録する（テスト用、templ.Component使用時）
	SentEmails []SendInput
	// SentRawEmails はレンダリング済み文字列で送信されたメールを記録する（テスト用、Worker経由時）
	SentRawEmails []SendRawInput
}

// NewNoopSender は新しいNoopSenderを作成する
func NewNoopSender() *NoopSender {
	return &NoopSender{
		SentEmails:    make([]SendInput, 0),
		SentRawEmails: make([]SendRawInput, 0),
	}
}

// Send はメールを送信せず、記録のみ行う（templ.Component使用時）
func (s *NoopSender) Send(_ context.Context, input SendInput) error {
	s.SentEmails = append(s.SentEmails, input)
	return nil
}

// SendRaw はレンダリング済み文字列でのメール送信を記録する（Worker経由時）
func (s *NoopSender) SendRaw(_ context.Context, input SendRawInput) error {
	s.SentRawEmails = append(s.SentRawEmails, input)
	return nil
}
