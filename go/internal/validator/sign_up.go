package validator

import (
	"context"
	"net/mail"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// SignUpCreateValidatorはサインアップフォームのバリデーションを行う
type SignUpCreateValidator struct {
	userRepo *repository.UserRepository
}

// NewSignUpCreateValidatorはSignUpCreateValidatorを生成する
func NewSignUpCreateValidator(userRepo *repository.UserRepository) *SignUpCreateValidator {
	return &SignUpCreateValidator{
		userRepo: userRepo,
	}
}

// SignUpCreateValidatorInputはバリデーションの入力パラメータ
type SignUpCreateValidatorInput struct {
	Email string
}

// Validateは入力値をチェックする (形式チェック + DB検証)
func (v *SignUpCreateValidator) Validate(ctx context.Context, input SignUpCreateValidatorInput) error {
	ve := model.NewValidationError()

	// メールアドレスの必須チェック
	if input.Email == "" {
		ve.AddField("email", i18n.T(ctx, "validation_required"))
		return ve
	}

	// メールアドレス形式チェック
	if _, err := mail.ParseAddress(input.Email); err != nil {
		ve.AddField("email", i18n.T(ctx, "validation_email_invalid"))
		return ve
	}

	// メールアドレスの重複チェック (DB検証)
	exists, err := v.userRepo.ExistsByEmail(ctx, input.Email)
	if err != nil {
		return err
	}
	if exists {
		ve.AddField("email", i18n.T(ctx, "validation_email_already_taken"))
		return ve
	}

	return nil
}
