package email

import (
	"context"
	"fmt"

	"github.com/a-h/templ"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/templates/emails/email_confirmation"
)

// ConfirmationSender はメール確認コードの送信を担当する
// テンプレートレンダリングとi18nによる件名取得をemailパッケージ内に閉じ込める
type ConfirmationSender struct {
	sender Sender
}

// NewConfirmationSender は新しいConfirmationSenderを作成する
func NewConfirmationSender(sender Sender) *ConfirmationSender {
	return &ConfirmationSender{sender: sender}
}

// Send はメール確認コードをレンダリングして送信する
func (s *ConfirmationSender) Send(ctx context.Context, to, code, locale string) error {
	ctx = i18n.SetLocale(ctx, locale)
	subject := i18n.T(ctx, "email_confirmation_subject")

	var htmlBody, textBody templ.Component
	switch locale {
	case "en":
		htmlBody = email_confirmation.EnHTML(to, code)
		textBody = email_confirmation.EnText(to, code)
	default:
		htmlBody = email_confirmation.JaHTML(to, code)
		textBody = email_confirmation.JaText(to, code)
	}

	if err := s.sender.Send(ctx, SendInput{
		To:       to,
		Subject:  subject,
		HTMLBody: htmlBody,
		TextBody: textBody,
	}); err != nil {
		return fmt.Errorf("メール確認コードの送信に失敗: %w", err)
	}

	return nil
}
