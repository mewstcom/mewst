package usecase

import (
	"context"

	"github.com/mewstcom/mewst/go/internal/auth"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// UpdatePasswordInput はパスワード更新の入力データ
type UpdatePasswordInput struct {
	Email    string
	Password string
}

// UpdatePasswordUsecase はパスワード更新のユースケース
type UpdatePasswordUsecase struct {
	userRepo *repository.UserRepository
}

// NewUpdatePasswordUsecase は新しいUpdatePasswordUsecaseを作成する
func NewUpdatePasswordUsecase(userRepo *repository.UserRepository) *UpdatePasswordUsecase {
	return &UpdatePasswordUsecase{
		userRepo: userRepo,
	}
}

// Execute はパスワード更新を実行する
func (uc *UpdatePasswordUsecase) Execute(ctx context.Context, input UpdatePasswordInput) error {
	// パスワードをbcryptでハッシュ化
	passwordDigest, err := auth.HashPassword(input.Password)
	if err != nil {
		return err
	}

	// パスワードを更新
	return uc.userRepo.UpdatePasswordByEmail(ctx, input.Email, passwordDigest)
}
