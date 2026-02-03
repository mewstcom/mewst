package repository_test

import (
	"context"
	"testing"

	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

func TestUserRepository_GetByID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	// テストユーザーを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("getbyid@example.com").
		WithLocale("ja").
		WithTimeZone("Asia/Tokyo").
		Build()

	repo := repository.NewUserRepository(tx)

	t.Run("存在するユーザーを取得できる", func(t *testing.T) {
		user, err := repo.GetByID(ctx, userID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}

		if user.ID != userID {
			t.Errorf("user.ID = %v, want %v", user.ID, userID)
		}
		if user.Email != "getbyid@example.com" {
			t.Errorf("user.Email = %v, want getbyid@example.com", user.Email)
		}
		if user.Locale != "ja" {
			t.Errorf("user.Locale = %v, want ja", user.Locale)
		}
		if user.TimeZone != "Asia/Tokyo" {
			t.Errorf("user.TimeZone = %v, want Asia/Tokyo", user.TimeZone)
		}
	})

	t.Run("存在しないユーザーはErrNotFoundを返す", func(t *testing.T) {
		nonExistentID := testutil.NewUserBuilder(t, tx).Build()
		// 作成したユーザーを削除して存在しない状態にする
		_, err := tx.Exec("DELETE FROM users WHERE id = $1", nonExistentID)
		if err != nil {
			t.Fatalf("ユーザー削除に失敗: %v", err)
		}

		_, err = repo.GetByID(ctx, nonExistentID)
		if err == nil {
			t.Error("GetByID() should return error for non-existent user")
		}
		if err != repository.ErrNotFound {
			t.Errorf("GetByID() error = %v, want ErrNotFound", err)
		}
	})
}

func TestUserRepository_GetByEmail(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	// テストユーザーを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("getbyemail@example.com").
		Build()

	repo := repository.NewUserRepository(tx)

	t.Run("存在するユーザーをメールアドレスで取得できる", func(t *testing.T) {
		user, err := repo.GetByEmail(ctx, "getbyemail@example.com")
		if err != nil {
			t.Fatalf("GetByEmail() error = %v", err)
		}

		if user.ID != userID {
			t.Errorf("user.ID = %v, want %v", user.ID, userID)
		}
		if user.Email != "getbyemail@example.com" {
			t.Errorf("user.Email = %v, want getbyemail@example.com", user.Email)
		}
	})

	t.Run("存在しないメールアドレスはErrNotFoundを返す", func(t *testing.T) {
		_, err := repo.GetByEmail(ctx, "nonexistent@example.com")
		if err == nil {
			t.Error("GetByEmail() should return error for non-existent email")
		}
		if err != repository.ErrNotFound {
			t.Errorf("GetByEmail() error = %v, want ErrNotFound", err)
		}
	})
}

func TestUserRepository_GetByEmailForSignIn(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	// テストユーザーを作成（パスワードダイジェストを指定）
	passwordDigest := "$2a$12$TestPasswordDigest123456789012345678901234567890"
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("signin@example.com").
		WithPasswordDigest(passwordDigest).
		Build()

	repo := repository.NewUserRepository(tx)

	t.Run("ログイン用にユーザーを取得できる", func(t *testing.T) {
		user, err := repo.GetByEmailForSignIn(ctx, "signin@example.com")
		if err != nil {
			t.Fatalf("GetByEmailForSignIn() error = %v", err)
		}

		if user.ID != userID {
			t.Errorf("user.ID = %v, want %v", user.ID, userID)
		}
		if user.Email != "signin@example.com" {
			t.Errorf("user.Email = %v, want signin@example.com", user.Email)
		}
		if user.PasswordDigest != passwordDigest {
			t.Errorf("user.PasswordDigest = %v, want %v", user.PasswordDigest, passwordDigest)
		}
	})

	t.Run("存在しないメールアドレスはErrNotFoundを返す", func(t *testing.T) {
		_, err := repo.GetByEmailForSignIn(ctx, "nonexistent@example.com")
		if err == nil {
			t.Error("GetByEmailForSignIn() should return error for non-existent email")
		}
		if err != repository.ErrNotFound {
			t.Errorf("GetByEmailForSignIn() error = %v, want ErrNotFound", err)
		}
	})
}
