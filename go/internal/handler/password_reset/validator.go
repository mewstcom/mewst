package password_reset

import (
	"context"
	"net/mail"

	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/templates"
)

// CreateValidator はパスワードリセット開始フォームの入力データと形式バリデーション
type CreateValidator struct {
	Email string
}

// Validate は入力値の形式をチェックする（DBアクセスなし）
// 形式バリデーションのみを行い、ユーザー存在チェックは行わない
func (v *CreateValidator) Validate(ctx context.Context) *session.FormErrors {
	errors := session.NewFormErrors()

	// メールアドレスの必須チェック
	if v.Email == "" {
		errors.AddFieldError("email", templates.T(ctx, "error_required"))
	} else {
		// メールアドレス形式チェック
		if _, err := mail.ParseAddress(v.Email); err != nil {
			errors.AddFieldError("email", templates.T(ctx, "error_invalid_email"))
		}
	}

	return errors
}
