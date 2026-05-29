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

// Execute はパスワード更新を実行する。
// バリデーション → ハッシュ化 → UPDATE の 1 ステップ書き込みで完結するため、
// オーケストレーションすべき対象がなく Execute 内で完結させている。
func (uc *UpdatePasswordUsecase) Execute(ctx context.Context, input UpdatePasswordInput) error {
	if err := uc.passwordValidator.Validate(ctx, validator.PasswordUpdateValidatorInput{
		Password: input.Password,
	}); err != nil {
		return err
	}

	passwordDigest, err := auth.HashPassword(input.Password)
	if err != nil {
		return fmt.Errorf("パスワードのハッシュ化に失敗: %w", err)
	}

	if err := uc.userRepo.UpdatePasswordByEmail(ctx, input.Email, passwordDigest); err != nil {
		return fmt.Errorf("パスワードの更新に失敗: %w", err)
	}
	return nil
}
