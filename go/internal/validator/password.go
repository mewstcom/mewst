package validator

import (
	"context"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/model"
)

// PasswordUpdateValidator はパスワード更新フォームのバリデーションを行う
type PasswordUpdateValidator struct{}

// NewPasswordUpdateValidator はPasswordUpdateValidatorを生成する
func NewPasswordUpdateValidator() *PasswordUpdateValidator {
	return &PasswordUpdateValidator{}
}

// PasswordUpdateValidatorInput はバリデーションの入力パラメータ
type PasswordUpdateValidatorInput struct {
	Password string
}

// PasswordUpdateValidatorOutput はバリデーション成功時の出力
type PasswordUpdateValidatorOutput struct{}

// Validate は入力値の形式をチェックする（DBアクセスなし）
func (v *PasswordUpdateValidator) Validate(ctx context.Context, input PasswordUpdateValidatorInput) (*PasswordUpdateValidatorOutput, error) {
	ve := model.NewValidationError()

	// パスワードの必須チェック
	if input.Password == "" {
		ve.AddField("password", i18n.T(ctx, "error_required"))
		return nil, ve
	}

	// 最小文字数チェック（8文字以上）
	if len([]rune(input.Password)) < 8 {
		ve.AddField("password", i18n.T(ctx, "error_password_too_short"))
		return nil, ve
	}

	// 最大バイト数チェック（72バイト以下、bcrypt制限）
	if len(input.Password) > 72 {
		ve.AddField("password", i18n.T(ctx, "error_password_too_long"))
		return nil, ve
	}

	return &PasswordUpdateValidatorOutput{}, nil
}
