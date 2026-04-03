package usecase

import (
	"context"
	"fmt"

	"github.com/mewstcom/mewst/go/internal/auth"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/validator"
)

// UpdatePasswordInput はパスワード更新の入力データ
type UpdatePasswordInput struct {
	Email    string
	Password string
}

// UpdatePasswordUsecase はパスワード更新のユースケース
type UpdatePasswordUsecase struct {
	passwordValidator *validator.PasswordUpdateValidator
	userRepo          *repository.UserRepository
}

// NewUpdatePasswordUsecase は新しいUpdatePasswordUsecaseを作成する
func NewUpdatePasswordUsecase(
	passwordValidator *validator.PasswordUpdateValidator,
	userRepo *repository.UserRepository,
) *UpdatePasswordUsecase {
	return &UpdatePasswordUsecase{
		passwordValidator: passwordValidator,
		userRepo:          userRepo,
	}
}

// Execute はパスワード更新を実行する
func (uc *UpdatePasswordUsecase) Execute(ctx context.Context, input UpdatePasswordInput) error {
	// 1. バリデーション
	_, err := uc.passwordValidator.Validate(ctx, validator.PasswordUpdateValidatorInput{
		Password: input.Password,
	})
	if err != nil {
		return err
	}

	// 2. パスワードをbcryptでハッシュ化
	passwordDigest, err := auth.HashPassword(input.Password)
	if err != nil {
		return fmt.Errorf("パスワードのハッシュ化に失敗: %w", err)
	}

	// 3. パスワードを更新
	return uc.userRepo.UpdatePasswordByEmail(ctx, input.Email, passwordDigest)
}
