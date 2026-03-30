// Package validator はフォーム入力のバリデーションを提供します
package validator

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

// SignInCreateValidator はサインインのバリデーションを行う
type SignInCreateValidator struct {
	userRepo *repository.UserRepository
}

// NewSignInCreateValidator はSignInCreateValidatorを生成する
func NewSignInCreateValidator(userRepo *repository.UserRepository) *SignInCreateValidator {
	return &SignInCreateValidator{userRepo: userRepo}
}

// SignInCreateValidatorInput はバリデーションの入力パラメータ
type SignInCreateValidatorInput struct {
	Email    string
	Password string
}

// SignInCreateValidatorResult はバリデーションの結果
type SignInCreateValidatorResult struct {
	User       *model.User
	FormErrors *session.FormErrors
	Err        error
}

// Validate はバリデーションを行う
func (v *SignInCreateValidator) Validate(ctx context.Context, input SignInCreateValidatorInput) *SignInCreateValidatorResult {
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
		return &SignInCreateValidatorResult{FormErrors: formErrors}
	}

	// 2. 状態バリデーション（DB検証）
	user, err := v.userRepo.GetByEmailForSignIn(ctx, input.Email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			formErrors.AddGlobalError(templates.T(ctx, "error_invalid_credentials"))
			return &SignInCreateValidatorResult{FormErrors: formErrors}
		}
		return &SignInCreateValidatorResult{Err: err}
	}

	// パスワードを検証
	if err := auth.CheckPassword(user.PasswordDigest, input.Password); err != nil {
		formErrors.AddGlobalError(templates.T(ctx, "error_invalid_credentials"))
		return &SignInCreateValidatorResult{FormErrors: formErrors}
	}

	return &SignInCreateValidatorResult{User: user}
}
