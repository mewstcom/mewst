package password

import (
	"context"

	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/templates"
)

// UpdateValidator はパスワード更新フォームの入力データと形式バリデーション
type UpdateValidator struct {
	Password string
}

// Validate は入力値の形式をチェックする（DBアクセスなし）
func (v *UpdateValidator) Validate(ctx context.Context) *session.FormErrors {
	errors := session.NewFormErrors()

	// パスワードの必須チェック
	if v.Password == "" {
		errors.AddFieldError("password", templates.T(ctx, "error_required"))
		return errors
	}

	// 最小文字数チェック（8文字以上）
	if len([]rune(v.Password)) < 8 {
		errors.AddFieldError("password", templates.T(ctx, "error_password_too_short"))
		return errors
	}

	// 最大バイト数チェック（72バイト以下、bcrypt制限）
	if len(v.Password) > 72 {
		errors.AddFieldError("password", templates.T(ctx, "error_password_too_long"))
	}

	return errors
}
