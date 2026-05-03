package repository_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

func TestUserProfileRepository_FindByUserID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	// テストデータを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("userprofile-user@example.com").
		Build()
	profileID := testutil.NewProfileBuilder(t, tx).
		WithAtname("userprofileuser").
		Build()

	// user_profilesを作成
	var userProfileID string
	err := tx.QueryRow(`
		INSERT INTO user_profiles (user_id, profile_id, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		RETURNING id
	`, uuid.UUID(userID), uuid.UUID(profileID)).Scan(&userProfileID)
	if err != nil {
		t.Fatalf("user_profiles作成に失敗: %v", err)
	}

	repo := repository.NewUserProfileRepository(testutil.QueriesWithTx(tx))

	t.Run("存在するユーザープロフィールをユーザーIDで取得できる", func(t *testing.T) {
		userProfile, err := repo.FindByUserID(ctx, userID)
		if err != nil {
			t.Fatalf("FindByUserID() error = %v", err)
		}

		if userProfile.UserID != userID {
			t.Errorf("userProfile.UserID = %v, want %v", userProfile.UserID, userID)
		}
		if userProfile.ProfileID != profileID {
			t.Errorf("userProfile.ProfileID = %v, want %v", userProfile.ProfileID, profileID)
		}
	})

	t.Run("存在しないユーザーIDはnilを返す", func(t *testing.T) {
		nonExistentUserID := testutil.NewUserBuilder(t, tx).Build()

		userProfile, err := repo.FindByUserID(ctx, nonExistentUserID)
		if err != nil {
			t.Errorf("FindByUserID() error = %v, want nil", err)
		}
		if userProfile != nil {
			t.Errorf("FindByUserID() userProfile = %v, want nil", userProfile)
		}
	})
}

func TestUserProfileRepository_FindByProfileID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	// テストデータを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("userprofile-profile@example.com").
		Build()
	profileID := testutil.NewProfileBuilder(t, tx).
		WithAtname("userprofileprofile").
		Build()

	// user_profilesを作成
	_, err := tx.Exec(`
		INSERT INTO user_profiles (user_id, profile_id, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
	`, uuid.UUID(userID), uuid.UUID(profileID))
	if err != nil {
		t.Fatalf("user_profiles作成に失敗: %v", err)
	}

	repo := repository.NewUserProfileRepository(testutil.QueriesWithTx(tx))

	t.Run("存在するユーザープロフィールをプロフィールIDで取得できる", func(t *testing.T) {
		userProfile, err := repo.FindByProfileID(ctx, profileID)
		if err != nil {
			t.Fatalf("FindByProfileID() error = %v", err)
		}

		if userProfile.UserID != userID {
			t.Errorf("userProfile.UserID = %v, want %v", userProfile.UserID, userID)
		}
		if userProfile.ProfileID != profileID {
			t.Errorf("userProfile.ProfileID = %v, want %v", userProfile.ProfileID, profileID)
		}
	})

	t.Run("存在しないプロフィールIDはnilを返す", func(t *testing.T) {
		nonExistentProfileID := testutil.NewProfileBuilder(t, tx).Build()

		userProfile, err := repo.FindByProfileID(ctx, nonExistentProfileID)
		if err != nil {
			t.Errorf("FindByProfileID() error = %v, want nil", err)
		}
		if userProfile != nil {
			t.Errorf("FindByProfileID() userProfile = %v, want nil", userProfile)
		}
	})
}

func TestUserProfileRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	// テストデータを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("userprofile-create@example.com").
		Build()
	profileID := testutil.NewProfileBuilder(t, tx).
		WithAtname("userprofilecreate").
		Build()

	repo := repository.NewUserProfileRepository(testutil.QueriesWithTx(tx))

	t.Run("ユーザープロフィール関連付けを作成できる", func(t *testing.T) {
		userProfile, err := repo.Create(ctx, repository.CreateUserProfileInput{
			UserID:    userID,
			ProfileID: profileID,
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if userProfile.UserID != userID {
			t.Errorf("userProfile.UserID = %v, want %v", userProfile.UserID, userID)
		}
		if userProfile.ProfileID != profileID {
			t.Errorf("userProfile.ProfileID = %v, want %v", userProfile.ProfileID, profileID)
		}
	})
}

func TestUserProfileRepository_WithTx(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	repo := repository.NewUserProfileRepository(testutil.QueriesWithTx(tx))

	// WithTxでトランザクションを設定したリポジトリを取得
	txRepo := repo.WithTx(tx)

	// テストデータを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("userprofile-withtx@example.com").
		Build()
	profileID := testutil.NewProfileBuilder(t, tx).
		WithAtname("userprofilewithtx").
		Build()

	t.Run("トランザクション内でユーザープロフィールを作成できる", func(t *testing.T) {
		userProfile, err := txRepo.Create(ctx, repository.CreateUserProfileInput{
			UserID:    userID,
			ProfileID: profileID,
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// 作成したユーザープロフィールを取得できることを確認
		fetched, err := txRepo.FindByUserID(ctx, userProfile.UserID)
		if err != nil {
			t.Fatalf("FindByUserID() error = %v", err)
		}
		if fetched.ProfileID != profileID {
			t.Errorf("fetched.ProfileID = %v, want %v", fetched.ProfileID, profileID)
		}
	})
}
