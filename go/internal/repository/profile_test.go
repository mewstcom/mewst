package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

func TestProfileRepository_FindByID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	// テストプロフィールを作成
	profileID := testutil.NewProfileBuilder(t, tx).
		WithAtname("testuser").
		WithName("Test User").
		Build()

	repo := repository.NewProfileRepository(testutil.QueriesWithTx(tx))

	t.Run("存在するプロフィールを取得できる", func(t *testing.T) {
		profile, err := repo.FindByID(ctx, profileID)
		if err != nil {
			t.Fatalf("FindByID()のエラー = %v", err)
		}

		if profile.ID != profileID {
			t.Errorf("profile.ID = %v、期待値 = %v", profile.ID, profileID)
		}
		if profile.Atname != "testuser" {
			t.Errorf("profile.Atname = %v、期待値 = testuser", profile.Atname)
		}
		if profile.Name != "Test User" {
			t.Errorf("profile.Name = %v、期待値 = Test User", profile.Name)
		}
	})

	t.Run("存在しないプロフィールはnilを返す", func(t *testing.T) {
		nonExistentID := testutil.NewProfileBuilder(t, tx).Build()
		_, err := tx.Exec("DELETE FROM profiles WHERE id = $1", uuid.UUID(nonExistentID))
		if err != nil {
			t.Fatalf("プロフィール削除に失敗: %v", err)
		}

		profile, err := repo.FindByID(ctx, nonExistentID)
		if err != nil {
			t.Errorf("FindByID()のエラー = %v、期待値 = nil", err)
		}
		if profile != nil {
			t.Errorf("FindByID()のprofile = %v、期待値 = nil", profile)
		}
	})
}

func TestProfileRepository_FindByAtname(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	// テストプロフィールを作成
	profileID := testutil.NewProfileBuilder(t, tx).
		WithAtname("findbyatname").
		WithName("Find By Atname").
		Build()

	repo := repository.NewProfileRepository(testutil.QueriesWithTx(tx))

	t.Run("存在するプロフィールをアットネームで取得できる", func(t *testing.T) {
		profile, err := repo.FindByAtname(ctx, "findbyatname")
		if err != nil {
			t.Fatalf("FindByAtname()のエラー = %v", err)
		}

		if profile.ID != profileID {
			t.Errorf("profile.ID = %v、期待値 = %v", profile.ID, profileID)
		}
		if profile.Atname != "findbyatname" {
			t.Errorf("profile.Atname = %v、期待値 = findbyatname", profile.Atname)
		}
	})

	t.Run("存在しないアットネームはnilを返す", func(t *testing.T) {
		profile, err := repo.FindByAtname(ctx, "nonexistent")
		if err != nil {
			t.Errorf("FindByAtname()のエラー = %v、期待値 = nil", err)
		}
		if profile != nil {
			t.Errorf("FindByAtname()のprofile = %v、期待値 = nil", profile)
		}
	})
}

func TestProfileRepository_ExistsByAtname(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	// テストプロフィールを作成
	_ = testutil.NewProfileBuilder(t, tx).
		WithAtname("existstest").
		Build()

	repo := repository.NewProfileRepository(testutil.QueriesWithTx(tx))

	t.Run("存在するアットネームはtrueを返す", func(t *testing.T) {
		exists, err := repo.ExistsByAtname(ctx, "existstest")
		if err != nil {
			t.Fatalf("ExistsByAtname()のエラー = %v", err)
		}
		if !exists {
			t.Error("ExistsByAtname() = false、期待値 = true")
		}
	})

	t.Run("存在しないアットネームはfalseを返す", func(t *testing.T) {
		exists, err := repo.ExistsByAtname(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("ExistsByAtname()のエラー = %v", err)
		}
		if exists {
			t.Error("ExistsByAtname() = true、期待値 = false")
		}
	})
}

func TestProfileRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	repo := repository.NewProfileRepository(testutil.QueriesWithTx(tx))

	t.Run("プロフィールを作成できる", func(t *testing.T) {
		joinedAt := time.Now()
		profile, err := repo.Create(ctx, repository.CreateProfileInput{
			OwnerType:     "Actor",
			Atname:        "newuser",
			Name:          "New User",
			Description:   "This is a test user",
			ImageURL:      "",
			JoinedAt:      joinedAt,
			AvatarKind:    "default",
			GravatarEmail: "",
			GravatarURL:   "",
		})
		if err != nil {
			t.Fatalf("Create()のエラー = %v", err)
		}

		if profile.Atname != "newuser" {
			t.Errorf("profile.Atname = %v、期待値 = newuser", profile.Atname)
		}
		if profile.Name != "New User" {
			t.Errorf("profile.Name = %v、期待値 = New User", profile.Name)
		}
		if profile.Description != "This is a test user" {
			t.Errorf("profile.Description = %v、期待値 = This is a test user", profile.Description)
		}
		if profile.OwnerType != "Actor" {
			t.Errorf("profile.OwnerType = %v、期待値 = Actor", profile.OwnerType)
		}
		if profile.AvatarKind != "default" {
			t.Errorf("profile.AvatarKind = %v、期待値 = default", profile.AvatarKind)
		}
	})
}

func TestProfileRepository_UpdateLastPostAt(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	profileID := testutil.NewProfileBuilder(t, tx).Build()

	repo := repository.NewProfileRepository(testutil.QueriesWithTx(tx))

	t.Run("last_post_atを更新できる", func(t *testing.T) {
		// 作成直後のlast_post_atはNULLであることを確認する。
		before, err := repo.FindByID(ctx, profileID)
		if err != nil {
			t.Fatalf("FindByID()のエラー = %v", err)
		}
		if before.LastPostAt != nil {
			t.Errorf("before.LastPostAt = %v、期待値 = nil", before.LastPostAt)
		}

		lastPostAt := time.Date(2024, 6, 7, 8, 9, 10, 0, time.UTC)
		if err := repo.UpdateLastPostAt(ctx, profileID, lastPostAt); err != nil {
			t.Fatalf("UpdateLastPostAt()のエラー = %v", err)
		}

		after, err := repo.FindByID(ctx, profileID)
		if err != nil {
			t.Fatalf("FindByID()のエラー = %v", err)
		}
		if after.LastPostAt == nil {
			t.Fatal("after.LastPostAt = nil、非nilを期待")
		}
		if !after.LastPostAt.Equal(lastPostAt) {
			t.Errorf("after.LastPostAt = %v、期待値 = %v", after.LastPostAt, lastPostAt)
		}
	})
}

func TestProfileRepository_WithTx(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	repo := repository.NewProfileRepository(testutil.QueriesWithTx(tx))

	// WithTxでトランザクションを設定したリポジトリを取得
	txRepo := repo.WithTx(tx)

	t.Run("トランザクション内でプロフィールを作成できる", func(t *testing.T) {
		profile, err := txRepo.Create(ctx, repository.CreateProfileInput{
			OwnerType:     "Actor",
			Atname:        "txuser",
			Name:          "Transaction User",
			Description:   "",
			ImageURL:      "",
			JoinedAt:      time.Now(),
			AvatarKind:    "default",
			GravatarEmail: "",
			GravatarURL:   "",
		})
		if err != nil {
			t.Fatalf("Create()のエラー = %v", err)
		}

		// 作成したプロフィールを取得できることを確認
		fetched, err := txRepo.FindByID(ctx, profile.ID)
		if err != nil {
			t.Fatalf("FindByID()のエラー = %v", err)
		}
		if fetched.Atname != "txuser" {
			t.Errorf("fetched.Atname = %v、期待値 = txuser", fetched.Atname)
		}
	})
}
