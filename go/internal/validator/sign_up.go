package validator

import (
	"context"
	"net/mail"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
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

// SignUpCreateValidatorOutput はバリデーション成功時の出力
type SignUpCreateValidatorOutput struct{}

// Validate は入力値をチェックする（形式チェック + DB検証）
func (v *SignUpCreateValidator) Validate(ctx context.Context, input SignUpCreateValidatorInput) (*SignUpCreateValidatorOutput, error) {
	ve := model.NewValidationError()

	// メールアドレスの必須チェック
	if input.Email == "" {
		ve.AddField("email", templates.T(ctx, "error_required"))
		return nil, ve
	}

	// メールアドレス形式チェック
	if _, err := mail.ParseAddress(input.Email); err != nil {
		ve.AddField("email", templates.T(ctx, "error_invalid_email"))
		return nil, ve
	}

	// メールアドレスの重複チェック（DB検証）
	exists, err := v.userRepo.ExistsByEmail(ctx, input.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		ve.AddField("email", templates.T(ctx, "error_email_already_taken"))
		return nil, ve
	}

	return &SignUpCreateValidatorOutput{}, nil
}
