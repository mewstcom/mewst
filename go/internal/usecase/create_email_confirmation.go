package usecase

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/a-h/templ"

	"github.com/mewstcom/mewst/internal/email"
	"github.com/mewstcom/mewst/internal/model"
	"github.com/mewstcom/mewst/internal/repository"
	email_confirmation_tmpl "github.com/mewstcom/mewst/internal/templates/emails/email_confirmation"
	"github.com/mewstcom/mewst/internal/worker"
)

// CreateEmailConfirmationUsecase はメール確認作成のユースケース
type CreateEmailConfirmationUsecase struct {
	emailConfirmRepo *repository.EmailConfirmationRepository
	emailSender      email.Sender
	workerClient     *worker.Client // nilの場合は同期送信
}

// NewCreateEmailConfirmationUsecase はCreateEmailConfirmationUsecaseを生成する
func NewCreateEmailConfirmationUsecase(
	emailConfirmRepo *repository.EmailConfirmationRepository,
	emailSender email.Sender,
) *CreateEmailConfirmationUsecase {
	return &CreateEmailConfirmationUsecase{
		emailConfirmRepo: emailConfirmRepo,
		emailSender:      emailSender,
		workerClient:     nil,
	}
}

// WithWorkerClient はWorkerクライアントを設定する（非同期メール送信用）
func (uc *CreateEmailConfirmationUsecase) WithWorkerClient(client *worker.Client) *CreateEmailConfirmationUsecase {
	uc.workerClient = client
	return uc
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
	subject := getEmailSubject(input.Locale)

	// Workerクライアントがある場合は非同期送信、ない場合は同期送信
	if uc.workerClient != nil {
		// テンプレートをレンダリング
		htmlStr, textStr, err := renderEmailTemplates(ctx, htmlBody, textBody)
		if err != nil {
			return nil, err
		}

		// ジョブをキューに追加
		_, err = uc.workerClient.Insert(ctx, worker.SendEmailArgs{
			To:       input.Email,
			Subject:  subject,
			HTMLBody: htmlStr,
			TextBody: textStr,
		})
		if err != nil {
			return nil, fmt.Errorf("メール送信ジョブの登録に失敗: %w", err)
		}
	} else {
		// 同期でメール送信
		if err := uc.emailSender.Send(ctx, email.SendInput{
			To:       input.Email,
			Subject:  subject,
			HTMLBody: htmlBody,
			TextBody: textBody,
		}); err != nil {
			return nil, fmt.Errorf("確認メールの送信に失敗: %w", err)
		}
	}

	return &CreateEmailConfirmationResult{
		EmailConfirmation: ec,
	}, nil
}

// renderEmailTemplates はメールテンプレートをレンダリングして文字列に変換する
func renderEmailTemplates(ctx context.Context, htmlBody, textBody templ.Component) (string, string, error) {
	var htmlBuf bytes.Buffer
	if err := htmlBody.Render(ctx, &htmlBuf); err != nil {
		return "", "", fmt.Errorf("HTMLテンプレートのレンダリングに失敗: %w", err)
	}

	var textBuf bytes.Buffer
	if err := textBody.Render(ctx, &textBuf); err != nil {
		return "", "", fmt.Errorf("テキストテンプレートのレンダリングに失敗: %w", err)
	}

	return htmlBuf.String(), textBuf.String(), nil
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
