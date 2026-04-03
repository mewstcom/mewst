package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/validator"
)

// VerifyEmailConfirmationUsecase はメール確認コード検証のユースケース
type VerifyEmailConfirmationUsecase struct {
	emailConfirmationValidator *validator.EmailConfirmationCreateValidator
	emailConfirmRepo           *repository.EmailConfirmationRepository
}

// NewVerifyEmailConfirmationUsecase は VerifyEmailConfirmationUsecase を生成する
func NewVerifyEmailConfirmationUsecase(
	emailConfirmationValidator *validator.EmailConfirmationCreateValidator,
	emailConfirmRepo *repository.EmailConfirmationRepository,
) *VerifyEmailConfirmationUsecase {
	return &VerifyEmailConfirmationUsecase{
		emailConfirmationValidator: emailConfirmationValidator,
		emailConfirmRepo:           emailConfirmRepo,
	}
}

// VerifyEmailConfirmationInput はメール確認コード検証の入力パラメータ
type VerifyEmailConfirmationInput struct {
	ID   uuid.UUID
	Code string
}

// VerifyEmailConfirmationOutput はメール確認コード検証の出力パラメータ
type VerifyEmailConfirmationOutput struct {
	EmailConfirmation *model.EmailConfirmation
}

// Execute はメール確認コード検証を実行する
func (uc *VerifyEmailConfirmationUsecase) Execute(ctx context.Context, input VerifyEmailConfirmationInput) (*VerifyEmailConfirmationOutput, error) {
	// 1. バリデーション（形式チェック + コード一致チェック）
	validateOutput, err := uc.emailConfirmationValidator.Validate(ctx, validator.EmailConfirmationCreateValidatorInput{
		ID:   input.ID,
		Code: input.Code,
	})
	if err != nil {
		return nil, err
	}

	// 2. メール確認を成功としてマーク
	if err := uc.emailConfirmRepo.MarkAsSucceeded(ctx, input.ID); err != nil {
		return nil, fmt.Errorf("メール確認の成功マークに失敗: %w", err)
	}

	return &VerifyEmailConfirmationOutput{
		EmailConfirmation: validateOutput.EmailConfirmation,
	}, nil
}
