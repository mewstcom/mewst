package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

func TestProfileRepository_GetByID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	// テストプロフィールを作成
	profileID := testutil.NewProfileBuilder(t, tx).
		WithAtname("testuser").
		WithName("Test User").
		Build()

	repo := repository.NewProfileRepository(tx)

	t.Run("存在するプロフィールを取得できる", func(t *testing.T) {
		profile, err := repo.GetByID(ctx, profileID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}

		if profile.ID != profileID {
			t.Errorf("profile.ID = %v, want %v", profile.ID, profileID)
		}
		if profile.Atname != "testuser" {
			t.Errorf("profile.Atname = %v, want testuser", profile.Atname)
		}
		if profile.Name != "Test User" {
			t.Errorf("profile.Name = %v, want Test User", profile.Name)
		}
	})

	t.Run("存在しないプロフィールはErrNotFoundを返す", func(t *testing.T) {
		nonExistentID := testutil.NewProfileBuilder(t, tx).Build()
		_, err := tx.Exec("DELETE FROM profiles WHERE id = $1", nonExistentID)
		if err != nil {
			t.Fatalf("プロフィール削除に失敗: %v", err)
		}

		_, err = repo.GetByID(ctx, nonExistentID)
		if err == nil {
			t.Error("GetByID() should return error for non-existent profile")
		}
		if err != repository.ErrNotFound {
			t.Errorf("GetByID() error = %v, want ErrNotFound", err)
		}
	})
}

func TestProfileRepository_GetByAtname(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	// テストプロフィールを作成
	profileID := testutil.NewProfileBuilder(t, tx).
		WithAtname("findbyatname").
		WithName("Find By Atname").
		Build()

	repo := repository.NewProfileRepository(tx)

	t.Run("存在するプロフィールをアットネームで取得できる", func(t *testing.T) {
		profile, err := repo.GetByAtname(ctx, "findbyatname")
		if err != nil {
			t.Fatalf("GetByAtname() error = %v", err)
		}

		if profile.ID != profileID {
			t.Errorf("profile.ID = %v, want %v", profile.ID, profileID)
		}
		if profile.Atname != "findbyatname" {
			t.Errorf("profile.Atname = %v, want findbyatname", profile.Atname)
		}
	})

	t.Run("存在しないアットネームはErrNotFoundを返す", func(t *testing.T) {
		_, err := repo.GetByAtname(ctx, "nonexistent")
		if err == nil {
			t.Error("GetByAtname() should return error for non-existent atname")
		}
		if err != repository.ErrNotFound {
			t.Errorf("GetByAtname() error = %v, want ErrNotFound", err)
		}
	})
}

func TestProfileRepository_ExistsByAtname(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	// テストプロフィールを作成
	_ = testutil.NewProfileBuilder(t, tx).
		WithAtname("existstest").
		Build()

	repo := repository.NewProfileRepository(tx)

	t.Run("存在するアットネームはtrueを返す", func(t *testing.T) {
		exists, err := repo.ExistsByAtname(ctx, "existstest")
		if err != nil {
			t.Fatalf("ExistsByAtname() error = %v", err)
		}
		if !exists {
			t.Error("ExistsByAtname() = false, want true")
		}
	})

	t.Run("存在しないアットネームはfalseを返す", func(t *testing.T) {
		exists, err := repo.ExistsByAtname(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("ExistsByAtname() error = %v", err)
		}
		if exists {
			t.Error("ExistsByAtname() = true, want false")
		}
	})
}

func TestProfileRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	repo := repository.NewProfileRepository(tx)

	t.Run("プロフィールを作成できる", func(t *testing.T) {
		joinedAt := time.Now()
		profile, err := repo.Create(ctx, repository.CreateProfileParams{
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
			t.Fatalf("Create() error = %v", err)
		}

		if profile.Atname != "newuser" {
			t.Errorf("profile.Atname = %v, want newuser", profile.Atname)
		}
		if profile.Name != "New User" {
			t.Errorf("profile.Name = %v, want New User", profile.Name)
		}
		if profile.Description != "This is a test user" {
			t.Errorf("profile.Description = %v, want This is a test user", profile.Description)
		}
		if profile.OwnerType != "Actor" {
			t.Errorf("profile.OwnerType = %v, want Actor", profile.OwnerType)
		}
		if profile.AvatarKind != "default" {
			t.Errorf("profile.AvatarKind = %v, want default", profile.AvatarKind)
		}
	})
}

func TestProfileRepository_WithTx(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	repo := repository.NewProfileRepository(tx)

	// WithTxでトランザクションを設定したリポジトリを取得
	txRepo := repo.WithTx(tx)

	t.Run("トランザクション内でプロフィールを作成できる", func(t *testing.T) {
		profile, err := txRepo.Create(ctx, repository.CreateProfileParams{
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
			t.Fatalf("Create() error = %v", err)
		}

		// 作成したプロフィールを取得できることを確認
		fetched, err := txRepo.GetByID(ctx, profile.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}
		if fetched.Atname != "txuser" {
			t.Errorf("fetched.Atname = %v, want txuser", fetched.Atname)
		}
	})
}
