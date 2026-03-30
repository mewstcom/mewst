package validator

import (
	"context"

	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/templates"
)

// PasswordUpdateValidator はパスワード更新フォームのバリデーションを行う
type PasswordUpdateValidator struct{}

// NewPasswordUpdateValidator はPasswordUpdateValidatorを生成する
func NewPasswordUpdateValidator() *PasswordUpdateValidator {
	return &PasswordUpdateValidator{}
}

// PasswordUpdateValidatorInput はバリデーションの入力パラメータ
type PasswordUpdateValidatorInput struct {
	Password string
}

// PasswordUpdateValidatorResult はバリデーションの結果
type PasswordUpdateValidatorResult struct {
	FormErrors *session.FormErrors
}

// Validate は入力値の形式をチェックする（DBアクセスなし）
func (v *PasswordUpdateValidator) Validate(ctx context.Context, input PasswordUpdateValidatorInput) *PasswordUpdateValidatorResult {
	formErrors := session.NewFormErrors()

	// パスワードの必須チェック
	if input.Password == "" {
		formErrors.AddFieldError("password", templates.T(ctx, "error_required"))
		return &PasswordUpdateValidatorResult{FormErrors: formErrors}
	}

	// 最小文字数チェック（8文字以上）
	if len([]rune(input.Password)) < 8 {
		formErrors.AddFieldError("password", templates.T(ctx, "error_password_too_short"))
		return &PasswordUpdateValidatorResult{FormErrors: formErrors}
	}

	// 最大バイト数チェック（72バイト以下、bcrypt制限）
	if len(input.Password) > 72 {
		formErrors.AddFieldError("password", templates.T(ctx, "error_password_too_long"))
	}

	return &PasswordUpdateValidatorResult{FormErrors: formErrors}
}
