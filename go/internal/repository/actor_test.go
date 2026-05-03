package repository_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

func TestActorRepository_FindByID(t *testing.T) {
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

	repo := repository.NewActorRepository(testutil.QueriesWithTx(tx))

	t.Run("存在するアクターを取得できる", func(t *testing.T) {
		actor, err := repo.FindByID(ctx, actorID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
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

	t.Run("存在しないアクターはnilを返す", func(t *testing.T) {
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
		_, err := tx.Exec("DELETE FROM actors WHERE id = $1", uuid.UUID(tempActorID))
		if err != nil {
			t.Fatalf("アクター削除に失敗: %v", err)
		}

		actor, err := repo.FindByID(ctx, tempActorID)
		if err != nil {
			t.Errorf("FindByID() error = %v, want nil", err)
		}
		if actor != nil {
			t.Errorf("FindByID() actor = %v, want nil", actor)
		}
	})
}

func TestActorRepository_FindByUserID(t *testing.T) {
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

	repo := repository.NewActorRepository(testutil.QueriesWithTx(tx))

	t.Run("存在するアクターをユーザーIDで取得できる", func(t *testing.T) {
		actor, err := repo.FindByUserID(ctx, userID)
		if err != nil {
			t.Fatalf("FindByUserID() error = %v", err)
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

	t.Run("存在しないユーザーIDはnilを返す", func(t *testing.T) {
		// アクターを持たないユーザーを作成
		noActorUserID := testutil.NewUserBuilder(t, tx).
			WithEmail("noactor@example.com").
			Build()

		actor, err := repo.FindByUserID(ctx, noActorUserID)
		if err != nil {
			t.Errorf("FindByUserID() error = %v, want nil", err)
		}
		if actor != nil {
			t.Errorf("FindByUserID() actor = %v, want nil", actor)
		}
	})
}
