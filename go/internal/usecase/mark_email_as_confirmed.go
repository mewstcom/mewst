package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// MarkEmailAsConfirmedUsecase はメール確認を完了としてマークするユースケース
type MarkEmailAsConfirmedUsecase struct {
	emailConfirmRepo *repository.EmailConfirmationRepository
}

// NewMarkEmailAsConfirmedUsecase はMarkEmailAsConfirmedUsecaseを生成する
func NewMarkEmailAsConfirmedUsecase(
	emailConfirmRepo *repository.EmailConfirmationRepository,
) *MarkEmailAsConfirmedUsecase {
	return &MarkEmailAsConfirmedUsecase{
		emailConfirmRepo: emailConfirmRepo,
	}
}

// MarkEmailAsConfirmedInput はメール確認完了の入力パラメータ
type MarkEmailAsConfirmedInput struct {
	EmailConfirmationID uuid.UUID
}

// MarkEmailAsConfirmedResult はメール確認完了の結果
type MarkEmailAsConfirmedResult struct {
	EmailConfirmation *model.EmailConfirmation
}

// Execute はメール確認を完了としてマークする（永続化のみ）
func (uc *MarkEmailAsConfirmedUsecase) Execute(ctx context.Context, input MarkEmailAsConfirmedInput) (*MarkEmailAsConfirmedResult, error) {
	// 確認を成功としてマーク
	if err := uc.emailConfirmRepo.MarkAsSucceeded(ctx, input.EmailConfirmationID); err != nil {
		return nil, fmt.Errorf("メール確認の成功マークに失敗: %w", err)
	}

	// 更新後のレコードを取得
	updatedEC, err := uc.emailConfirmRepo.GetByID(ctx, input.EmailConfirmationID)
	if err != nil {
		return nil, fmt.Errorf("更新後のメール確認レコードの取得に失敗: %w", err)
	}

	return &MarkEmailAsConfirmedResult{
		EmailConfirmation: updatedEC,
	}, nil
}
