package validator

import (
	"context"
	"net/mail"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/model"
)

// PasswordResetCreateValidatorはパスワードリセット開始フォームのバリデーションを行う
type PasswordResetCreateValidator struct{}

// NewPasswordResetCreateValidatorはPasswordResetCreateValidatorを生成する
func NewPasswordResetCreateValidator() *PasswordResetCreateValidator {
	return &PasswordResetCreateValidator{}
}

// PasswordResetCreateValidatorInputはバリデーションの入力パラメータ
type PasswordResetCreateValidatorInput struct {
	Email string
}

// Validateは入力値の形式をチェックする (DBアクセスなし)
//
// メールアドレスの存在チェックを行わないのは、列挙攻撃 (存在するメールアドレスを
// 推測する攻撃) を防ぐため。存在しないメールでも「リセットメールを送信しました」
// と返すことで、攻撃者にメールアドレスの存在有無を推測させない。
func (v *PasswordResetCreateValidator) Validate(ctx context.Context, input PasswordResetCreateValidatorInput) error {
	ve := model.NewValidationError()

	// メールアドレスの必須チェック
	if input.Email == "" {
		ve.AddField("email", i18n.T(ctx, "validation_required"))
		return ve
	}

	// メールアドレス形式チェック
	if _, err := mail.ParseAddress(input.Email); err != nil {
		ve.AddField("email", i18n.T(ctx, "validation_email_invalid"))
	}

	if ve.HasErrors() {
		return ve
	}

	return nil
}
