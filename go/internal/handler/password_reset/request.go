package password_reset

import (
	"context"
	"net/mail"

	"github.com/mewstcom/mewst/internal/session"
	"github.com/mewstcom/mewst/internal/templates"
)

// CreateRequest はパスワードリセット開始フォームのリクエストデータ
type CreateRequest struct {
	Email string
}

// Validate はリクエストデータを検証する
// 形式バリデーションのみを行い、ユーザー存在チェックは行わない
func (r *CreateRequest) Validate(ctx context.Context) *session.FormErrors {
	errors := session.NewFormErrors()

	// メールアドレスの必須チェック
	if r.Email == "" {
		errors.AddFieldError("email", templates.T(ctx, "error_required"))
	} else {
		// メールアドレス形式チェック
		if _, err := mail.ParseAddress(r.Email); err != nil {
			errors.AddFieldError("email", templates.T(ctx, "error_invalid_email"))
		}
	}

	return errors
}
