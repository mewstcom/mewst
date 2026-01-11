package usecase

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/a-h/templ"

	"github.com/mewstcom/mewst/internal/email"
	"github.com/mewstcom/mewst/internal/model"
	"github.com/mewstcom/mewst/internal/repository"
	email_confirmation_tmpl "github.com/mewstcom/mewst/internal/templates/emails/email_confirmation"
)

// CreateEmailConfirmationUsecase はメール確認作成のユースケース
type CreateEmailConfirmationUsecase struct {
	emailConfirmRepo *repository.EmailConfirmationRepository
	emailSender      email.Sender
}

// NewCreateEmailConfirmationUsecase はCreateEmailConfirmationUsecaseを生成する
func NewCreateEmailConfirmationUsecase(
	emailConfirmRepo *repository.EmailConfirmationRepository,
	emailSender email.Sender,
) *CreateEmailConfirmationUsecase {
	return &CreateEmailConfirmationUsecase{
		emailConfirmRepo: emailConfirmRepo,
		emailSender:      emailSender,
	}
}

// CreateEmailConfirmationInput はメール確認作成の入力パラメータ
type CreateEmailConfirmationInput struct {
	Email  string
	Event  model.EmailConfirmationEvent
	Locale string
}

// CreateEmailConfirmationResult はメール確認作成の結果
type CreateEmailConfirmationResult struct {
	EmailConfirmation *model.EmailConfirmation
}

// Execute はメール確認を作成し、確認メールを送信する
func (uc *CreateEmailConfirmationUsecase) Execute(ctx context.Context, input CreateEmailConfirmationInput) (*CreateEmailConfirmationResult, error) {
	// 6桁のランダムなコードを生成
	code, err := generateConfirmationCode()
	if err != nil {
		return nil, fmt.Errorf("確認コードの生成に失敗: %w", err)
	}

	// メール確認レコードを作成
	ec, err := uc.emailConfirmRepo.Create(ctx, repository.CreateEmailConfirmationParams{
		Email: input.Email,
		Event: input.Event,
		Code:  code,
	})
	if err != nil {
		return nil, fmt.Errorf("メール確認レコードの作成に失敗: %w", err)
	}

	// メールテンプレートを選択（ロケールに基づく）
	htmlBody, textBody := getEmailTemplates(input.Locale, input.Email, code)

	// 確認メールを送信
	if err := uc.emailSender.SendEmailConfirmation(ctx, email.SendEmailConfirmationInput{
		To:       input.Email,
		Subject:  getEmailSubject(input.Locale),
		HTMLBody: htmlBody,
		TextBody: textBody,
	}); err != nil {
		return nil, fmt.Errorf("確認メールの送信に失敗: %w", err)
	}

	return &CreateEmailConfirmationResult{
		EmailConfirmation: ec,
	}, nil
}

// generateConfirmationCode は6桁のランダムな数字コードを生成する
func generateConfirmationCode() (string, error) {
	// 000000 から 999999 までのランダムな数を生成
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	// 6桁になるようにゼロ埋め
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// getEmailTemplates はロケールに基づいてメールテンプレートを返す
func getEmailTemplates(locale, emailAddr, code string) (htmlBody, textBody templ.Component) {
	switch locale {
	case "en":
		return email_confirmation_tmpl.EnHTML(emailAddr, code), email_confirmation_tmpl.EnText(emailAddr, code)
	default:
		return email_confirmation_tmpl.JaHTML(emailAddr, code), email_confirmation_tmpl.JaText(emailAddr, code)
	}
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
