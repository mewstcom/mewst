package usecase

import (
	"context"
	"fmt"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// GetActiveEmailConfirmationUsecase は有効なメール確認を取得するユースケース
type GetActiveEmailConfirmationUsecase struct {
	emailConfirmationRepo *repository.EmailConfirmationRepository
}

// NewGetActiveEmailConfirmationUsecase はGetActiveEmailConfirmationUsecaseを生成する
func NewGetActiveEmailConfirmationUsecase(
	emailConfirmationRepo *repository.EmailConfirmationRepository,
) *GetActiveEmailConfirmationUsecase {
	return &GetActiveEmailConfirmationUsecase{
		emailConfirmationRepo: emailConfirmationRepo,
	}
}

// GetActiveEmailConfirmationInput は有効なメール確認取得の入力パラメータ
type GetActiveEmailConfirmationInput struct {
	ID model.EmailConfirmationID
}

// GetActiveEmailConfirmationOutput は有効なメール確認取得の結果
type GetActiveEmailConfirmationOutput struct {
	EmailConfirmation *model.EmailConfirmation
}

// Execute は有効期限内かつ未確認のメール確認を取得する
func (uc *GetActiveEmailConfirmationUsecase) Execute(ctx context.Context, input GetActiveEmailConfirmationInput) (*GetActiveEmailConfirmationOutput, error) {
	ec, err := uc.emailConfirmationRepo.FindActiveByID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("有効なメール確認の取得に失敗: %w", err)
	}
	if ec == nil {
		return nil, ErrNotFound
	}

	return &GetActiveEmailConfirmationOutput{EmailConfirmation: ec}, nil
}
