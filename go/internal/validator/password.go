package validator

import (
	"context"
	"unicode/utf8"

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

// Validate は入力値の形式をチェックする（DBアクセスなし）
func (v *PasswordUpdateValidator) Validate(ctx context.Context, input PasswordUpdateValidatorInput) error {
	ve := model.NewValidationError()

	// パスワードの必須チェック
	if input.Password == "" {
		ve.AddField("password", i18n.T(ctx, "validation_required"))
		return ve
	}

	// 最小文字数チェック
	if utf8.RuneCountInString(input.Password) < minPasswordLength {
		ve.AddField("password", i18n.T(ctx, "validation_password_too_short"))
		return ve
	}

	// 最大バイト数チェック（bcrypt 制限）
	if len(input.Password) > maxPasswordLength {
		ve.AddField("password", i18n.T(ctx, "validation_password_too_long"))
		return ve
	}

	return nil
}
