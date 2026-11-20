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

		_, tx := testutil.SetupTx(t)

		// テストユーザーを作成
		email := "test-update-password@example.com"
		testutil.NewUserBuilder(t, tx).
			WithEmail(email).
			Build()

		// Usecaseを作成
		userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
		passwordValidator := validator.NewPasswordUpdateValidator()
		uc := NewUpdatePasswordUsecase(passwordValidator, userRepo)

		// パスワードを更新
		newPassword := "newPassword123"
		err := uc.Execute(context.Background(), UpdatePasswordInput{
			Email:    email,
			Password: newPassword,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		// 更新後のパスワードで検証できることを確認
		user, err := userRepo.FindByEmail(context.Background(), email)
		if err != nil {
			t.Fatalf("FindByEmail()のエラー = %v", err)
		}

		err = auth.CheckPassword(user.PasswordDigest, newPassword)
		if err != nil {
			t.Errorf("新しいパスワードが有効であることを期待したが、無効だった: %v", err)
		}
	})

	t.Run("古いパスワードでは検証できなくなる", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)

		// テストユーザーを作成 (デフォルトパスワードは "password")
		email := "test-old-password@example.com"
		testutil.NewUserBuilder(t, tx).
			WithEmail(email).
			Build()

		// Usecaseを作成
		userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
		passwordValidator := validator.NewPasswordUpdateValidator()
		uc := NewUpdatePasswordUsecase(passwordValidator, userRepo)

		// パスワードを更新
		newPassword := "newPassword456"
		err := uc.Execute(context.Background(), UpdatePasswordInput{
			Email:    email,
			Password: newPassword,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		// 更新後のユーザーを取得
		user, err := userRepo.FindByEmail(context.Background(), email)
		if err != nil {
			t.Fatalf("FindByEmail()のエラー = %v", err)
		}

		// 古いパスワードでは検証できないことを確認
		oldPassword := "password"
		err = auth.CheckPassword(user.PasswordDigest, oldPassword)
		if err == nil {
			t.Error("古いパスワードが無効であることを期待したが、有効だった")
		}
	})

	t.Run("日本語を含むパスワードで更新できる", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)

		// テストユーザーを作成
		email := "test-japanese-password@example.com"
		testutil.NewUserBuilder(t, tx).
			WithEmail(email).
			Build()

		// Usecaseを作成
		userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
		passwordValidator := validator.NewPasswordUpdateValidator()
		uc := NewUpdatePasswordUsecase(passwordValidator, userRepo)

		// 日本語を含むパスワードで更新
		newPassword := "パスワード123abc"
		err := uc.Execute(context.Background(), UpdatePasswordInput{
			Email:    email,
			Password: newPassword,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		// 更新後のパスワードで検証できることを確認
		user, err := userRepo.FindByEmail(context.Background(), email)
		if err != nil {
			t.Fatalf("FindByEmail()のエラー = %v", err)
		}

		err = auth.CheckPassword(user.PasswordDigest, newPassword)
		if err != nil {
			t.Errorf("日本語のパスワードが有効であることを期待したが、無効だった: %v", err)
		}
	})
}
