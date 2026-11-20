// Package emailはメール送信機能を提供します
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

// Senderはメール送信を行うインターフェース
type Sender interface {
	// Sendはメールを送信する (templ.Componentを使用)
	Send(ctx context.Context, input SendInput) error
}

// SendInputはメール送信の入力 (templ.Componentを使用)
type SendInput struct {
	To       string          // 送信先メールアドレス
	Subject  string          // 件名
	HTMLBody templ.Component // メール本文 (HTML形式)
	TextBody templ.Component // メール本文 (テキスト形式)

	// IdempotencyKeyはメッセージに名前を与え、再送された送信が試行ごと
	// ではなく1通だけ配信されるようにする。API呼び出しは成功したがその結果を
	// 失った後に再試行されたジョブは、同じキーを使うため、プロバイダーは再送では
	// なく最初の配信を返す。
	//
	// 空のキーはキーを送らない。再試行が他の場所で既に守られている送信元は
	// これでよい。
	IdempotencyKey string
}

// ResendSenderはResend APIを使用してメールを送信する
type ResendSender struct {
	client    *resend.Client
	fromEmail string
	fromName  string
}

// NewResendSenderは新しいResendSenderを作成する
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

// fromはFromアドレスを生成する
// fromNameが設定されている場合は「名前 <メール>」形式、そうでない場合はメールアドレスのみ
func (s *ResendSender) from() string {
	if s.fromName != "" {
		return fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail)
	}
	return s.fromEmail
}

// Sendはメールを送信する (templ.Componentを使用)
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

	params := &resend.SendEmailRequest{
		From:    s.from(),
		To:      []string{input.To},
		Subject: input.Subject,
		Html:    htmlBuf.String(),
		Text:    textBuf.String(),
	}

	// Resendはキーが空でないときだけIdempotency-Keyヘッダーを送るため、
	// この1つの呼び出しでキーを設定しない送信元も扱える。
	options := &resend.SendEmailOptions{IdempotencyKey: input.IdempotencyKey}

	_, err := s.client.Emails.SendWithOptions(ctx, params, options)
	if err != nil {
		return fmt.Errorf("メール送信に失敗しました: %w", err)
	}

	return nil
}

// DiscardSenderはメールを配信・保持せずに送信を受け付ける。
// メールプロバイダーが未設定のときにruntimeで使用するsenderである。
type DiscardSender struct{}

// NewDiscardSenderはすべてのメールを破棄する無状態のsenderを作成する。
func NewDiscardSender() *DiscardSender {
	return &DiscardSender{}
}

// Sendは入力を一切保持せずにメールを破棄する。
func (s *DiscardSender) Send(_ context.Context, _ SendInput) error {
	return nil
}

// NoopSenderはメールを送信しないダミー実装 (テスト用)
type NoopSender struct {
	// SentEmailsは送信されたメールを記録する (テスト用)
	SentEmails []SendInput
}

// NewNoopSenderは新しいNoopSenderを作成する
func NewNoopSender() *NoopSender {
	return &NoopSender{
		SentEmails: make([]SendInput, 0),
	}
}

// Sendはメールを送信せず、記録のみ行う (テスト用)
func (s *NoopSender) Send(_ context.Context, input SendInput) error {
	s.SentEmails = append(s.SentEmails, input)
	return nil
}
