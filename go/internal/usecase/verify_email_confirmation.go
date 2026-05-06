package usecase

import (
	"context"
	"fmt"

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
	ID   model.EmailConfirmationID
	Code string
}

// VerifyEmailConfirmationOutput はメール確認コード検証の出力パラメータ
type VerifyEmailConfirmationOutput struct {
	EmailConfirmation *model.EmailConfirmation
}

// Execute はメール確認コード検証を実行する。
// バリデーション → 単一の永続化 (Succeed) で完結するため、
// オーケストレーションすべき対象がなく Execute 内で完結させている。
func (uc *VerifyEmailConfirmationUsecase) Execute(ctx context.Context, input VerifyEmailConfirmationInput) (*VerifyEmailConfirmationOutput, error) {
	emailConfirmation, err := uc.emailConfirmationValidator.Validate(ctx, validator.EmailConfirmationCreateValidatorInput{
		ID:   input.ID,
		Code: input.Code,
	})
	if err != nil {
		return nil, err
	}

	if err := uc.emailConfirmRepo.Succeed(ctx, emailConfirmation.ID); err != nil {
		return nil, fmt.Errorf("メール確認の成功マークに失敗: %w", err)
	}

	return &VerifyEmailConfirmationOutput{
		EmailConfirmation: emailConfirmation,
	}, nil
}
