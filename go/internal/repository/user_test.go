package repository_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

func TestUserRepository_FindByID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	// テストユーザーを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("findbyid@example.com").
		WithLocale("ja").
		WithTimeZone("Asia/Tokyo").
		Build()

	repo := repository.NewUserRepository(testutil.QueriesWithTx(tx))

	t.Run("存在するユーザーを取得できる", func(t *testing.T) {
		user, err := repo.FindByID(ctx, userID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}

		if user.ID != userID {
			t.Errorf("user.ID = %v, want %v", user.ID, userID)
		}
		if user.Email != "findbyid@example.com" {
			t.Errorf("user.Email = %v, want findbyid@example.com", user.Email)
		}
		if user.Locale != "ja" {
			t.Errorf("user.Locale = %v, want ja", user.Locale)
		}
		if user.TimeZone != "Asia/Tokyo" {
			t.Errorf("user.TimeZone = %v, want Asia/Tokyo", user.TimeZone)
		}
	})

	t.Run("存在しないユーザーはnilを返す", func(t *testing.T) {
		nonExistentID := testutil.NewUserBuilder(t, tx).Build()
		// 作成したユーザーを削除して存在しない状態にする
		_, err := tx.Exec("DELETE FROM users WHERE id = $1", uuid.UUID(nonExistentID))
		if err != nil {
			t.Fatalf("ユーザー削除に失敗: %v", err)
		}

		user, err := repo.FindByID(ctx, nonExistentID)
		if err != nil {
			t.Errorf("FindByID() error = %v, want nil", err)
		}
		if user != nil {
			t.Errorf("FindByID() user = %v, want nil", user)
		}
	})
}

func TestUserRepository_FindByEmail(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	// テストユーザーを作成（パスワードダイジェストも検証）
	passwordDigest := "$2a$12$TestPasswordDigest123456789012345678901234567890"
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("findbyemail@example.com").
		WithPasswordDigest(passwordDigest).
		Build()

	repo := repository.NewUserRepository(testutil.QueriesWithTx(tx))

	t.Run("存在するユーザーをメールアドレスで取得できる", func(t *testing.T) {
		user, err := repo.FindByEmail(ctx, "findbyemail@example.com")
		if err != nil {
			t.Fatalf("FindByEmail() error = %v", err)
		}

		if user.ID != userID {
			t.Errorf("user.ID = %v, want %v", user.ID, userID)
		}
		if user.Email != "findbyemail@example.com" {
			t.Errorf("user.Email = %v, want findbyemail@example.com", user.Email)
		}
		if user.PasswordDigest != passwordDigest {
			t.Errorf("user.PasswordDigest = %v, want %v", user.PasswordDigest, passwordDigest)
		}
	})

	t.Run("存在しないメールアドレスはnilを返す", func(t *testing.T) {
		user, err := repo.FindByEmail(ctx, "nonexistent@example.com")
		if err != nil {
			t.Errorf("FindByEmail() error = %v, want nil", err)
		}
		if user != nil {
			t.Errorf("FindByEmail() user = %v, want nil", user)
		}
	})
}
