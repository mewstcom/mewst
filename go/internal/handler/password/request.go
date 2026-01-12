package password

import (
	"context"

	"github.com/mewstcom/mewst/internal/session"
	"github.com/mewstcom/mewst/internal/templates"
)

// UpdateRequest はパスワード更新フォームのリクエストデータ
type UpdateRequest struct {
	Password string
}

// Validate はリクエストデータを検証する
func (r *UpdateRequest) Validate(ctx context.Context) *session.FormErrors {
	errors := session.NewFormErrors()

	// パスワードの必須チェック
	if r.Password == "" {
		errors.AddFieldError("password", templates.T(ctx, "error_required"))
		return errors
	}

	// 最小文字数チェック（8文字以上）
	if len([]rune(r.Password)) < 8 {
		errors.AddFieldError("password", templates.T(ctx, "error_password_too_short"))
		return errors
	}

	// 最大バイト数チェック（72バイト以下、bcrypt制限）
	if len(r.Password) > 72 {
		errors.AddFieldError("password", templates.T(ctx, "error_password_too_long"))
	}

	return errors
}
