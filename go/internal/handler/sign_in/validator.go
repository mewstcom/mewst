package sign_in

import (
	"context"
	"errors"
	"net/mail"

	"github.com/mewstcom/mewst/go/internal/auth"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/templates"
)

// CreateValidator はサインインのバリデーションを行う
type CreateValidator struct {
	userRepo *repository.UserRepository
}

// NewCreateValidator はCreateValidatorを生成する
func NewCreateValidator(userRepo *repository.UserRepository) *CreateValidator {
	return &CreateValidator{userRepo: userRepo}
}

// CreateValidatorInput はバリデーションの入力パラメータ
type CreateValidatorInput struct {
	Email    string
	Password string
}

// CreateValidatorResult はバリデーションの結果
type CreateValidatorResult struct {
	User       *model.User
	FormErrors *session.FormErrors
	Err        error
}

// Validate はバリデーションを行う
func (v *CreateValidator) Validate(ctx context.Context, input CreateValidatorInput) *CreateValidatorResult {
	// 1. 形式バリデーション
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

	// パスワードの必須チェック
	if input.Password == "" {
		formErrors.AddFieldError("password", templates.T(ctx, "error_required"))
	}

	if formErrors.HasErrors() {
		return &CreateValidatorResult{FormErrors: formErrors}
	}

	// 2. 状態バリデーション（DB検証）
	user, err := v.userRepo.GetByEmailForSignIn(ctx, input.Email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			formErrors.AddGlobalError(templates.T(ctx, "error_invalid_credentials"))
			return &CreateValidatorResult{FormErrors: formErrors}
		}
		return &CreateValidatorResult{Err: err}
	}

	// パスワードを検証
	if err := auth.CheckPassword(user.PasswordDigest, input.Password); err != nil {
		formErrors.AddGlobalError(templates.T(ctx, "error_invalid_credentials"))
		return &CreateValidatorResult{FormErrors: formErrors}
	}

	return &CreateValidatorResult{User: user}
}
