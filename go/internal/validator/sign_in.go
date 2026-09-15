// Package validatorはフォーム入力のバリデーションを提供します
package validator

import (
	"context"
	"net/mail"

	"github.com/mewstcom/mewst/go/internal/auth"
	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// SignInCreateValidatorはサインインのバリデーションを行う
type SignInCreateValidator struct {
	userRepo *repository.UserRepository
}

// NewSignInCreateValidatorはSignInCreateValidatorを生成する
func NewSignInCreateValidator(userRepo *repository.UserRepository) *SignInCreateValidator {
	return &SignInCreateValidator{userRepo: userRepo}
}

// SignInCreateValidatorInputはバリデーションの入力パラメータ
type SignInCreateValidatorInput struct {
	Email    string
	Password string
}

// Validateはバリデーションを行い、成功時はユーザーを返す
func (v *SignInCreateValidator) Validate(ctx context.Context, input SignInCreateValidatorInput) (*model.User, error) {
	ve := model.NewValidationError()

	// 1. 形式バリデーション
	v.validateEmail(ctx, ve, input.Email)
	v.validatePassword(ctx, ve, input.Password)

	if ve.HasErrors() {
		return nil, ve
	}

	// 2. 状態バリデーション (DB検証)
	user, err := v.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		return nil, err
	}
	// セキュリティ対策: 存在しないメールアドレスでも同じエラーメッセージを表示
	if user == nil {
		ve.AddGlobal(i18n.T(ctx, "validation_credentials_invalid"))
		return nil, ve
	}

	// パスワードを検証
	if err := auth.CheckPassword(user.PasswordDigest, input.Password); err != nil {
		ve.AddGlobal(i18n.T(ctx, "validation_credentials_invalid"))
		return nil, ve
	}

	return user, nil
}

// validateEmailはメールアドレスの形式バリデーションを行う
func (v *SignInCreateValidator) validateEmail(ctx context.Context, ve *model.ValidationError, email string) {
	if email == "" {
		ve.AddField("email", i18n.T(ctx, "validation_required"))
		return
	}

	if _, err := mail.ParseAddress(email); err != nil {
		ve.AddField("email", i18n.T(ctx, "validation_email_invalid"))
		return
	}
}

// validatePasswordはパスワードの形式バリデーションを行う
func (v *SignInCreateValidator) validatePassword(ctx context.Context, ve *model.ValidationError, password string) {
	if password == "" {
		ve.AddField("password", i18n.T(ctx, "validation_required"))
		return
	}
}
