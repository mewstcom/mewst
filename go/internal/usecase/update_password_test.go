package usecase

import (
	"context"
	"testing"

	"github.com/mewstcom/mewst/go/internal/auth"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/validator"
)

func TestUpdatePasswordUsecase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("パスワードを更新できる", func(t *testing.T) {
		t.Parallel()

		db, tx := testutil.SetupTestDB(t)

		// テストユーザーを作成
		email := "test-update-password@example.com"
		testutil.NewUserBuilder(t, tx).
			WithEmail(email).
			Build()

		// Usecaseを作成
		userRepo := repository.NewUserRepository(db).WithTx(tx)
		passwordValidator := validator.NewPasswordUpdateValidator()
		uc := NewUpdatePasswordUsecase(passwordValidator, userRepo)

		// パスワードを更新
		newPassword := "newPassword123"
		err := uc.Execute(context.Background(), UpdatePasswordInput{
			Email:    email,
			Password: newPassword,
		})
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		// 更新後のパスワードで検証できることを確認
		user, err := userRepo.GetByEmail(context.Background(), email)
		if err != nil {
			t.Fatalf("GetByEmail failed: %v", err)
		}

		err = auth.CheckPassword(user.PasswordDigest, newPassword)
		if err != nil {
			t.Errorf("new password should be valid: %v", err)
		}
	})

	t.Run("古いパスワードでは検証できなくなる", func(t *testing.T) {
		t.Parallel()

		db, tx := testutil.SetupTestDB(t)

		// テストユーザーを作成（デフォルトパスワードは "password"）
		email := "test-old-password@example.com"
		testutil.NewUserBuilder(t, tx).
			WithEmail(email).
			Build()

		// Usecaseを作成
		userRepo := repository.NewUserRepository(db).WithTx(tx)
		passwordValidator := validator.NewPasswordUpdateValidator()
		uc := NewUpdatePasswordUsecase(passwordValidator, userRepo)

		// パスワードを更新
		newPassword := "newPassword456"
		err := uc.Execute(context.Background(), UpdatePasswordInput{
			Email:    email,
			Password: newPassword,
		})
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		// 更新後のユーザーを取得
		user, err := userRepo.GetByEmail(context.Background(), email)
		if err != nil {
			t.Fatalf("GetByEmail failed: %v", err)
		}

		// 古いパスワードでは検証できないことを確認
		oldPassword := "password"
		err = auth.CheckPassword(user.PasswordDigest, oldPassword)
		if err == nil {
			t.Error("old password should not be valid")
		}
	})

	t.Run("日本語を含むパスワードで更新できる", func(t *testing.T) {
		t.Parallel()

		db, tx := testutil.SetupTestDB(t)

		// テストユーザーを作成
		email := "test-japanese-password@example.com"
		testutil.NewUserBuilder(t, tx).
			WithEmail(email).
			Build()

		// Usecaseを作成
		userRepo := repository.NewUserRepository(db).WithTx(tx)
		passwordValidator := validator.NewPasswordUpdateValidator()
		uc := NewUpdatePasswordUsecase(passwordValidator, userRepo)

		// 日本語を含むパスワードで更新
		newPassword := "パスワード123abc"
		err := uc.Execute(context.Background(), UpdatePasswordInput{
			Email:    email,
			Password: newPassword,
		})
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		// 更新後のパスワードで検証できることを確認
		user, err := userRepo.GetByEmail(context.Background(), email)
		if err != nil {
			t.Fatalf("GetByEmail failed: %v", err)
		}

		err = auth.CheckPassword(user.PasswordDigest, newPassword)
		if err != nil {
			t.Errorf("Japanese password should be valid: %v", err)
		}
	})
}
