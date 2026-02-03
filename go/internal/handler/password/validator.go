package password

import (
	"context"

	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/templates"
)

// UpdateValidator はパスワード更新フォームのバリデーションを行う
type UpdateValidator struct{}

// NewUpdateValidator はUpdateValidatorを生成する
func NewUpdateValidator() *UpdateValidator {
	return &UpdateValidator{}
}

// UpdateValidatorInput はバリデーションの入力パラメータ
type UpdateValidatorInput struct {
	Password string
}

// UpdateValidatorResult はバリデーションの結果
type UpdateValidatorResult struct {
	FormErrors *session.FormErrors
}

// Validate は入力値の形式をチェックする（DBアクセスなし）
func (v *UpdateValidator) Validate(ctx context.Context, input UpdateValidatorInput) *UpdateValidatorResult {
	formErrors := session.NewFormErrors()

	// パスワードの必須チェック
	if input.Password == "" {
		formErrors.AddFieldError("password", templates.T(ctx, "error_required"))
		return &UpdateValidatorResult{FormErrors: formErrors}
	}

	// 最小文字数チェック（8文字以上）
	if len([]rune(input.Password)) < 8 {
		formErrors.AddFieldError("password", templates.T(ctx, "error_password_too_short"))
		return &UpdateValidatorResult{FormErrors: formErrors}
	}

	// 最大バイト数チェック（72バイト以下、bcrypt制限）
	if len(input.Password) > 72 {
		formErrors.AddFieldError("password", templates.T(ctx, "error_password_too_long"))
	}

	return &UpdateValidatorResult{FormErrors: formErrors}
}
