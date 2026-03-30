package validator

import (
	"context"
	"net/mail"

	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/templates"
)

// SignUpCreateValidator はサインアップフォームのバリデーションを行う
type SignUpCreateValidator struct {
	userRepo *repository.UserRepository
}

// NewSignUpCreateValidator はSignUpCreateValidatorを生成する
func NewSignUpCreateValidator(userRepo *repository.UserRepository) *SignUpCreateValidator {
	return &SignUpCreateValidator{
		userRepo: userRepo,
	}
}

// SignUpCreateValidatorInput はバリデーションの入力パラメータ
type SignUpCreateValidatorInput struct {
	Email string
}

// SignUpCreateValidatorResult はバリデーションの結果
type SignUpCreateValidatorResult struct {
	FormErrors *session.FormErrors
	Err        error
}

// Validate は入力値をチェックする（形式チェック + DB検証）
func (v *SignUpCreateValidator) Validate(ctx context.Context, input SignUpCreateValidatorInput) *SignUpCreateValidatorResult {
	formErrors := session.NewFormErrors()

	// メールアドレスの必須チェック
	if input.Email == "" {
		formErrors.AddFieldError("email", templates.T(ctx, "error_required"))
		return &SignUpCreateValidatorResult{FormErrors: formErrors}
	}

	// メールアドレス形式チェック
	if _, err := mail.ParseAddress(input.Email); err != nil {
		formErrors.AddFieldError("email", templates.T(ctx, "error_invalid_email"))
		return &SignUpCreateValidatorResult{FormErrors: formErrors}
	}

	// メールアドレスの重複チェック（DB検証）
	exists, err := v.userRepo.ExistsByEmail(ctx, input.Email)
	if err != nil {
		return &SignUpCreateValidatorResult{Err: err}
	}
	if exists {
		formErrors.AddFieldError("email", templates.T(ctx, "error_email_already_taken"))
		return &SignUpCreateValidatorResult{FormErrors: formErrors}
	}

	return &SignUpCreateValidatorResult{FormErrors: formErrors}
}
