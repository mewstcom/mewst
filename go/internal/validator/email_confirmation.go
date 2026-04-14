package validator

import (
	"context"
	"errors"
	"regexp"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// 確認コードは6桁の数字
var codeRegex = regexp.MustCompile(`^\d{6}$`)

// EmailConfirmationCreateValidator はメール確認のバリデーションを行う
type EmailConfirmationCreateValidator struct {
	emailConfirmationRepo *repository.EmailConfirmationRepository
}

// NewEmailConfirmationCreateValidator はEmailConfirmationCreateValidatorを生成する
func NewEmailConfirmationCreateValidator(emailConfirmationRepo *repository.EmailConfirmationRepository) *EmailConfirmationCreateValidator {
	return &EmailConfirmationCreateValidator{emailConfirmationRepo: emailConfirmationRepo}
}

// EmailConfirmationCreateValidatorInput はバリデーションの入力パラメータ
type EmailConfirmationCreateValidatorInput struct {
	ID   uuid.UUID
	Code string
}

// EmailConfirmationCreateValidatorOutput はバリデーション成功時の出力
type EmailConfirmationCreateValidatorOutput struct {
	EmailConfirmation *model.EmailConfirmation
}

// Validate はバリデーションを行う
func (v *EmailConfirmationCreateValidator) Validate(ctx context.Context, input EmailConfirmationCreateValidatorInput) (*EmailConfirmationCreateValidatorOutput, error) {
	// 1. 形式バリデーション
	ve := model.NewValidationError()

	// 確認コードの必須チェック
	if input.Code == "" {
		ve.AddField("code", i18n.T(ctx, "error_required"))
	} else {
		// 6桁の数字形式チェック
		if !codeRegex.MatchString(input.Code) {
			ve.AddField("code", i18n.T(ctx, "error_invalid_code_format"))
		}
	}

	if ve.HasErrors() {
		return nil, ve
	}

	// 2. 状態バリデーション（DB検証）
	emailConfirmation, err := v.emailConfirmationRepo.GetActiveByID(ctx, input.ID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			ve.AddGlobal(i18n.T(ctx, "error_code_incorrect_or_expired"))
			return nil, ve
		}
		return nil, err
	}

	// 確認コードを検証
	if emailConfirmation.Code != input.Code {
		ve.AddGlobal(i18n.T(ctx, "error_code_incorrect_or_expired"))
		return nil, ve
	}

	return &EmailConfirmationCreateValidatorOutput{EmailConfirmation: emailConfirmation}, nil
}
