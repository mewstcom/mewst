package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// GetSucceededEmailConfirmationUsecase は確認済みのメール確認を取得するユースケース
type GetSucceededEmailConfirmationUsecase struct {
	emailConfirmationRepo *repository.EmailConfirmationRepository
}

// NewGetSucceededEmailConfirmationUsecase はGetSucceededEmailConfirmationUsecaseを生成する
func NewGetSucceededEmailConfirmationUsecase(
	emailConfirmationRepo *repository.EmailConfirmationRepository,
) *GetSucceededEmailConfirmationUsecase {
	return &GetSucceededEmailConfirmationUsecase{
		emailConfirmationRepo: emailConfirmationRepo,
	}
}

// GetSucceededEmailConfirmationInput は確認済みメール確認取得の入力パラメータ
type GetSucceededEmailConfirmationInput struct {
	ID uuid.UUID
}

// GetSucceededEmailConfirmationOutput は確認済みメール確認取得の結果
type GetSucceededEmailConfirmationOutput struct {
	EmailConfirmation *model.EmailConfirmation
}

// Execute は確認済みのメール確認を取得する
func (uc *GetSucceededEmailConfirmationUsecase) Execute(ctx context.Context, input GetSucceededEmailConfirmationInput) (*GetSucceededEmailConfirmationOutput, error) {
	ec, err := uc.emailConfirmationRepo.GetSucceededByID(ctx, input.ID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("確認済みメール確認の取得に失敗: %w", err)
	}

	return &GetSucceededEmailConfirmationOutput{EmailConfirmation: ec}, nil
}
