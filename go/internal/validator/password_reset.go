package validator

import (
	"context"
	"net/mail"

	"github.com/mewstcom/mewst/go/internal/model"
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

// PasswordResetCreateValidatorOutput はバリデーション成功時の出力
type PasswordResetCreateValidatorOutput struct{}

// Validate は入力値の形式をチェックする（DBアクセスなし）
// 形式バリデーションのみを行い、ユーザー存在チェックは行わない
func (v *PasswordResetCreateValidator) Validate(ctx context.Context, input PasswordResetCreateValidatorInput) (*PasswordResetCreateValidatorOutput, error) {
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

	if ve.HasErrors() {
		return nil, ve
	}

	return &PasswordResetCreateValidatorOutput{}, nil
}
