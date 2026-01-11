package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/internal/model"
	"github.com/mewstcom/mewst/internal/repository"
)

// ErrCodeMismatch は確認コードが一致しない場合のエラー
var ErrCodeMismatch = errors.New("確認コードが一致しません")

// ConfirmEmailUsecase はメール確認のユースケース
type ConfirmEmailUsecase struct {
	emailConfirmRepo *repository.EmailConfirmationRepository
}

// NewConfirmEmailUsecase はConfirmEmailUsecaseを生成する
func NewConfirmEmailUsecase(
	emailConfirmRepo *repository.EmailConfirmationRepository,
) *ConfirmEmailUsecase {
	return &ConfirmEmailUsecase{
		emailConfirmRepo: emailConfirmRepo,
	}
}

// ConfirmEmailInput はメール確認の入力パラメータ
type ConfirmEmailInput struct {
	EmailConfirmationID uuid.UUID
	Code                string
}

// ConfirmEmailResult はメール確認の結果
type ConfirmEmailResult struct {
	EmailConfirmation *model.EmailConfirmation
}

// Execute は確認コードを検証し、成功した場合はレコードを更新する
func (uc *ConfirmEmailUsecase) Execute(ctx context.Context, input ConfirmEmailInput) (*ConfirmEmailResult, error) {
	// 有効な確認レコードを取得（有効期限内かつ未確認）
	ec, err := uc.emailConfirmRepo.GetActiveByID(ctx, input.EmailConfirmationID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("メール確認レコードの取得に失敗: %w", err)
	}

	// 確認コードを検証
	if ec.Code != input.Code {
		return nil, ErrCodeMismatch
	}

	// 確認を成功としてマーク
	if err := uc.emailConfirmRepo.MarkAsSucceeded(ctx, input.EmailConfirmationID); err != nil {
		return nil, fmt.Errorf("メール確認の成功マークに失敗: %w", err)
	}

	// 更新後のレコードを取得
	updatedEC, err := uc.emailConfirmRepo.GetByID(ctx, input.EmailConfirmationID)
	if err != nil {
		return nil, fmt.Errorf("更新後のメール確認レコードの取得に失敗: %w", err)
	}

	return &ConfirmEmailResult{
		EmailConfirmation: updatedEC,
	}, nil
}
