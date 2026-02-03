package email_confirmation

import (
	"context"
	"errors"
	"regexp"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/templates"
)

// 確認コードは6桁の数字
var codeRegex = regexp.MustCompile(`^\d{6}$`)

// CreateValidator はメール確認のバリデーションを行う
type CreateValidator struct {
	emailConfirmationRepo *repository.EmailConfirmationRepository
}

// NewCreateValidator はCreateValidatorを生成する
func NewCreateValidator(emailConfirmationRepo *repository.EmailConfirmationRepository) *CreateValidator {
	return &CreateValidator{emailConfirmationRepo: emailConfirmationRepo}
}

// CreateValidatorInput はバリデーションの入力パラメータ
type CreateValidatorInput struct {
	ID   uuid.UUID
	Code string
}

// CreateValidatorResult はバリデーションの結果
type CreateValidatorResult struct {
	EmailConfirmation *model.EmailConfirmation
	FormErrors        *session.FormErrors
	Err               error
}

// Validate はバリデーションを行う
func (v *CreateValidator) Validate(ctx context.Context, input CreateValidatorInput) *CreateValidatorResult {
	// 1. 形式バリデーション
	formErrors := session.NewFormErrors()

	// 確認コードの必須チェック
	if input.Code == "" {
		formErrors.AddFieldError("code", templates.T(ctx, "error_required"))
	} else {
		// 6桁の数字形式チェック
		if !codeRegex.MatchString(input.Code) {
			formErrors.AddFieldError("code", templates.T(ctx, "error_invalid_code_format"))
		}
	}

	if formErrors.HasErrors() {
		return &CreateValidatorResult{FormErrors: formErrors}
	}

	// 2. 状態バリデーション（DB検証）
	emailConfirmation, err := v.emailConfirmationRepo.GetActiveByID(ctx, input.ID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			formErrors.AddGlobalError(templates.T(ctx, "error_code_incorrect_or_expired"))
			return &CreateValidatorResult{FormErrors: formErrors}
		}
		return &CreateValidatorResult{Err: err}
	}

	// 確認コードを検証
	if emailConfirmation.Code != input.Code {
		formErrors.AddGlobalError(templates.T(ctx, "error_code_incorrect_or_expired"))
		return &CreateValidatorResult{FormErrors: formErrors}
	}

	return &CreateValidatorResult{EmailConfirmation: emailConfirmation}
}
