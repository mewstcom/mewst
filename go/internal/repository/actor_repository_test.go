package repository_test

import (
	"context"
	"testing"

	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

func TestActorRepository_GetByID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	// テストデータを作成（User → Profile → Actor）
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("actor-getbyid@example.com").
		Build()
	profileID := testutil.NewProfileBuilder(t, tx).
		WithAtname("actoruser1").
		Build()
	actorID := testutil.NewActorBuilder(t, tx).
		WithUserID(userID).
		WithProfileID(profileID).
		Build()

	repo := repository.NewActorRepository(tx)

	t.Run("存在するアクターを取得できる", func(t *testing.T) {
		actor, err := repo.GetByID(ctx, actorID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}

		if actor.ID != actorID {
			t.Errorf("actor.ID = %v, want %v", actor.ID, actorID)
		}
		if actor.UserID != userID {
			t.Errorf("actor.UserID = %v, want %v", actor.UserID, userID)
		}
		if actor.ProfileID != profileID {
			t.Errorf("actor.ProfileID = %v, want %v", actor.ProfileID, profileID)
		}
	})

	t.Run("存在しないアクターはErrNotFoundを返す", func(t *testing.T) {
		// 一時的にアクターを作成して削除
		tempUserID := testutil.NewUserBuilder(t, tx).
			WithEmail("actor-temp@example.com").
			Build()
		tempProfileID := testutil.NewProfileBuilder(t, tx).
			WithAtname("tempactor").
			Build()
		tempActorID := testutil.NewActorBuilder(t, tx).
			WithUserID(tempUserID).
			WithProfileID(tempProfileID).
			Build()

		// 削除
		_, err := tx.Exec("DELETE FROM actors WHERE id = $1", tempActorID)
		if err != nil {
			t.Fatalf("アクター削除に失敗: %v", err)
		}

		_, err = repo.GetByID(ctx, tempActorID)
		if err == nil {
			t.Error("GetByID() should return error for non-existent actor")
		}
		if err != repository.ErrNotFound {
			t.Errorf("GetByID() error = %v, want ErrNotFound", err)
		}
	})
}

func TestActorRepository_GetByUserID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	// テストデータを作成（User → Profile → Actor）
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("actor-getbyuserid@example.com").
		Build()
	profileID := testutil.NewProfileBuilder(t, tx).
		WithAtname("actoruser2").
		Build()
	actorID := testutil.NewActorBuilder(t, tx).
		WithUserID(userID).
		WithProfileID(profileID).
		Build()

	repo := repository.NewActorRepository(tx)

	t.Run("存在するアクターをユーザーIDで取得できる", func(t *testing.T) {
		actor, err := repo.GetByUserID(ctx, userID)
		if err != nil {
			t.Fatalf("GetByUserID() error = %v", err)
		}

		if actor.ID != actorID {
			t.Errorf("actor.ID = %v, want %v", actor.ID, actorID)
		}
		if actor.UserID != userID {
			t.Errorf("actor.UserID = %v, want %v", actor.UserID, userID)
		}
		if actor.ProfileID != profileID {
			t.Errorf("actor.ProfileID = %v, want %v", actor.ProfileID, profileID)
		}
	})

	t.Run("存在しないユーザーIDはErrNotFoundを返す", func(t *testing.T) {
		// アクターを持たないユーザーを作成
		noActorUserID := testutil.NewUserBuilder(t, tx).
			WithEmail("noactor@example.com").
			Build()

		_, err := repo.GetByUserID(ctx, noActorUserID)
		if err == nil {
			t.Error("GetByUserID() should return error for user without actor")
		}
		if err != repository.ErrNotFound {
			t.Errorf("GetByUserID() error = %v, want ErrNotFound", err)
		}
	})
}
