package sign_in

import (
	"context"
	"net/mail"

	"github.com/mewstcom/mewst/internal/session"
	"github.com/mewstcom/mewst/internal/templates"
)

// CreateRequest はログインフォームのリクエストデータ
type CreateRequest struct {
	Email    string
	Password string
}

// Validate はリクエストデータを検証する
// 形式バリデーションのみを行い、認証チェックは行わない
func (r *CreateRequest) Validate(ctx context.Context) *session.FormErrors {
	errors := session.NewFormErrors()

	// メールアドレスの必須チェック
	if r.Email == "" {
		errors.AddFieldError("email", templates.T(ctx, "errors.validation.required", "email"))
	} else {
		// メールアドレス形式チェック
		if _, err := mail.ParseAddress(r.Email); err != nil {
			errors.AddFieldError("email", templates.T(ctx, "errors.validation.invalid_email"))
		}
	}

	// パスワードの必須チェック
	if r.Password == "" {
		errors.AddFieldError("password", templates.T(ctx, "errors.validation.required", "password"))
	}

	return errors
}
