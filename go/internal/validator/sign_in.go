// Package validator はフォーム入力のバリデーションを提供します
package validator

import (
	"context"
	"errors"
	"net/mail"

	"github.com/mewstcom/mewst/go/internal/auth"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
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

// SignInCreateValidatorOutput はバリデーション成功時の出力
type SignInCreateValidatorOutput struct {
	User *model.User
}

// Validate はバリデーションを行う
func (v *SignInCreateValidator) Validate(ctx context.Context, input SignInCreateValidatorInput) (*SignInCreateValidatorOutput, error) {
	// 1. 形式バリデーション
	ve := model.NewValidationError()

	// メールアドレスの必須チェック
	if input.Email == "" {
		ve.AddField("email", templates.T(ctx, "error_required"))
	} else {
		// メールアドレス形式チェック
		if _, err := mail.ParseAddress(input.Email); err != nil {
			ve.AddField("email", templates.T(ctx, "error_invalid_email"))
		}
	}

	// パスワードの必須チェック
	if input.Password == "" {
		ve.AddField("password", templates.T(ctx, "error_required"))
	}

	if ve.HasErrors() {
		return nil, ve
	}

	// 2. 状態バリデーション（DB検証）
	user, err := v.userRepo.GetByEmailForSignIn(ctx, input.Email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			// セキュリティ対策: 存在しないメールアドレスでも同じエラーメッセージを表示
			ve.AddGlobal(templates.T(ctx, "error_invalid_credentials"))
			return nil, ve
		}
		return nil, err
	}

	// パスワードを検証
	if err := auth.CheckPassword(user.PasswordDigest, input.Password); err != nil {
		ve.AddGlobal(templates.T(ctx, "error_invalid_credentials"))
		return nil, ve
	}

	return &SignInCreateValidatorOutput{User: user}, nil
}
