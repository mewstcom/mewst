package validator

import (
	"context"
	"net/mail"

	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/templates"
)

// PasswordResetCreateValidator はパスワードリセット開始フォームのバリデーションを行う
type PasswordResetCreateValidator struct{}

// NewPasswordResetCreateValidator はPasswordResetCreateValidatorを生成する
func NewPasswordResetCreateValidator() *PasswordResetCreateValidator {
	return &PasswordResetCreateValidator{}
}

// PasswordResetCreateValidatorInput はバリデーションの入力パラメータ
type PasswordResetCreateValidatorInput struct {
	Email string
}

// PasswordResetCreateValidatorResult はバリデーションの結果
type PasswordResetCreateValidatorResult struct {
	FormErrors *session.FormErrors
}

// Validate は入力値の形式をチェックする（DBアクセスなし）
// 形式バリデーションのみを行い、ユーザー存在チェックは行わない
func (v *PasswordResetCreateValidator) Validate(ctx context.Context, input PasswordResetCreateValidatorInput) *PasswordResetCreateValidatorResult {
	formErrors := session.NewFormErrors()

	// メールアドレスの必須チェック
	if input.Email == "" {
		formErrors.AddFieldError("email", templates.T(ctx, "error_required"))
	} else {
		// メールアドレス形式チェック
		if _, err := mail.ParseAddress(input.Email); err != nil {
			formErrors.AddFieldError("email", templates.T(ctx, "error_invalid_email"))
		}
	}

	return &PasswordResetCreateValidatorResult{FormErrors: formErrors}
}
