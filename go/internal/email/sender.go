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
	// Send はメールを送信する (templ.Componentを使用)
	Send(ctx context.Context, input SendInput) error
}

// SendInput はメール送信の入力 (templ.Componentを使用)
type SendInput struct {
	To       string          // 送信先メールアドレス
	Subject  string          // 件名
	HTMLBody templ.Component // メール本文 (HTML形式)
	TextBody templ.Component // メール本文 (テキスト形式)

	// IdempotencyKey names the message so that a retried send delivers one
	// email instead of one per attempt. A job that is retried after the API
	// call succeeded but its result was lost reuses the same key and the
	// provider returns the first delivery instead of sending again.
	//
	// An empty key sends no key, which is what senders whose retries are
	// already guarded elsewhere want.
	//
	// [Ja] IdempotencyKey はメッセージに名前を与え、再送された送信が試行ごと
	// ではなく 1 通だけ配信されるようにする。API 呼び出しは成功したがその結果を
	// 失った後に再試行されたジョブは、同じキーを使うため、プロバイダーは再送では
	// なく最初の配信を返す。
	//
	// 空のキーはキーを送らない。再試行が他の場所で既に守られている送信元は
	// これでよい。
	IdempotencyKey string
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

// Send はメールを送信する (templ.Componentを使用)
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

	// Resend sends the Idempotency-Key header only when the key is non-empty,
	// so this one call also covers senders that do not set one.
	//
	// [Ja] Resend はキーが空でないときだけ Idempotency-Key ヘッダーを送るため、
	// この 1 つの呼び出しでキーを設定しない送信元も扱える。
	options := &resend.SendEmailOptions{IdempotencyKey: input.IdempotencyKey}

	_, err := s.client.Emails.SendWithOptions(ctx, params, options)
	if err != nil {
		return fmt.Errorf("メール送信に失敗しました: %w", err)
	}

	return nil
}

// DiscardSender accepts email sends without delivering or retaining them.
// It is the runtime sender used when the email provider is not configured.
//
// [Ja] DiscardSender はメールを配信・保持せずに送信を受け付ける。
// メールプロバイダーが未設定のときに runtime で使用する sender である。
type DiscardSender struct{}

// NewDiscardSender creates a stateless sender that discards every email.
//
// [Ja] NewDiscardSender はすべてのメールを破棄する無状態の sender を作成する。
func NewDiscardSender() *DiscardSender {
	return &DiscardSender{}
}

// Send discards the email without retaining any part of the input.
//
// [Ja] Send は入力を一切保持せずにメールを破棄する。
func (s *DiscardSender) Send(_ context.Context, _ SendInput) error {
	return nil
}

// NoopSender はメールを送信しないダミー実装 (テスト用)
type NoopSender struct {
	// SentEmails は送信されたメールを記録する (テスト用)
	SentEmails []SendInput
}

// NewNoopSender は新しいNoopSenderを作成する
func NewNoopSender() *NoopSender {
	return &NoopSender{
		SentEmails: make([]SendInput, 0),
	}
}

// Send はメールを送信せず、記録のみ行う (テスト用)
func (s *NoopSender) Send(_ context.Context, input SendInput) error {
	s.SentEmails = append(s.SentEmails, input)
	return nil
}
