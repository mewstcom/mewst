package password_reset

import (
	"context"
	"net/mail"

	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/templates"
)

// CreateValidator はパスワードリセット開始フォームのバリデーションを行う
type CreateValidator struct{}

// NewCreateValidator はCreateValidatorを生成する
func NewCreateValidator() *CreateValidator {
	return &CreateValidator{}
}

// CreateValidatorInput はバリデーションの入力パラメータ
type CreateValidatorInput struct {
	Email string
}

// CreateValidatorResult はバリデーションの結果
type CreateValidatorResult struct {
	FormErrors *session.FormErrors
}

// Validate は入力値の形式をチェックする（DBアクセスなし）
// 形式バリデーションのみを行い、ユーザー存在チェックは行わない
func (v *CreateValidator) Validate(ctx context.Context, input CreateValidatorInput) *CreateValidatorResult {
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

	return &CreateValidatorResult{FormErrors: formErrors}
}
