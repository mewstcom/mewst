package sign_up

import (
	"context"
	"net/mail"

	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/templates"
)

// CreateValidator はサインアップフォームのバリデーションを行う
type CreateValidator struct {
	userRepo *repository.UserRepository
}

// NewCreateValidator はCreateValidatorを生成する
func NewCreateValidator(userRepo *repository.UserRepository) *CreateValidator {
	return &CreateValidator{
		userRepo: userRepo,
	}
}

// CreateValidatorInput はバリデーションの入力パラメータ
type CreateValidatorInput struct {
	Email string
}

// CreateValidatorResult はバリデーションの結果
type CreateValidatorResult struct {
	FormErrors *session.FormErrors
	Err        error
}

// Validate は入力値をチェックする（形式チェック + DB検証）
func (v *CreateValidator) Validate(ctx context.Context, input CreateValidatorInput) *CreateValidatorResult {
	formErrors := session.NewFormErrors()

	// メールアドレスの必須チェック
	if input.Email == "" {
		formErrors.AddFieldError("email", templates.T(ctx, "error_required"))
		return &CreateValidatorResult{FormErrors: formErrors}
	}

	// メールアドレス形式チェック
	if _, err := mail.ParseAddress(input.Email); err != nil {
		formErrors.AddFieldError("email", templates.T(ctx, "error_invalid_email"))
		return &CreateValidatorResult{FormErrors: formErrors}
	}

	// メールアドレスの重複チェック（DB検証）
	exists, err := v.userRepo.ExistsByEmail(ctx, input.Email)
	if err != nil {
		return &CreateValidatorResult{Err: err}
	}
	if exists {
		formErrors.AddFieldError("email", templates.T(ctx, "error_email_already_taken"))
		return &CreateValidatorResult{FormErrors: formErrors}
	}

	return &CreateValidatorResult{FormErrors: formErrors}
}
