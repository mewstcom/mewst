package repository_test

import (
	"context"
	"testing"

	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

func TestSessionRepository_GetByToken(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	// テストデータを作成（User → Profile → Actor → Session）
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("session-getbytoken@example.com").
		Build()
	profileID := testutil.NewProfileBuilder(t, tx).
		WithAtname("sessionuser1").
		Build()
	actorID := testutil.NewActorBuilder(t, tx).
		WithUserID(userID).
		WithProfileID(profileID).
		Build()

	token := "test-session-token-getbytoken"
	_ = testutil.NewSessionBuilder(t, tx).
		WithActorID(actorID).
		WithToken(token).
		WithIPAddress("192.168.1.1").
		WithUserAgent("Test Browser/1.0").
		Build()

	repo := repository.NewSessionRepository(tx)

	t.Run("存在するセッションをトークンで取得できる", func(t *testing.T) {
		session, err := repo.GetByToken(ctx, token)
		if err != nil {
			t.Fatalf("GetByToken() error = %v", err)
		}

		if session.Token != token {
			t.Errorf("session.Token = %v, want %v", session.Token, token)
		}
		if session.ActorID != actorID {
			t.Errorf("session.ActorID = %v, want %v", session.ActorID, actorID)
		}
		if session.IPAddress != "192.168.1.1" {
			t.Errorf("session.IPAddress = %v, want 192.168.1.1", session.IPAddress)
		}
		if session.UserAgent != "Test Browser/1.0" {
			t.Errorf("session.UserAgent = %v, want Test Browser/1.0", session.UserAgent)
		}
	})

	t.Run("存在しないトークンはErrNotFoundを返す", func(t *testing.T) {
		_, err := repo.GetByToken(ctx, "nonexistent-token")
		if err == nil {
			t.Error("GetByToken() should return error for non-existent token")
		}
		if err != repository.ErrNotFound {
			t.Errorf("GetByToken() error = %v, want ErrNotFound", err)
		}
	})
}

func TestSessionRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	// テストデータを作成（User → Profile → Actor）
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("session-create@example.com").
		Build()
	profileID := testutil.NewProfileBuilder(t, tx).
		WithAtname("sessionuser2").
		Build()
	actorID := testutil.NewActorBuilder(t, tx).
		WithUserID(userID).
		WithProfileID(profileID).
		Build()

	repo := repository.NewSessionRepository(tx)

	t.Run("セッションを作成できる", func(t *testing.T) {
		params := repository.CreateSessionParams{
			ActorID:   actorID,
			Token:     "new-session-token-create",
			IPAddress: "10.0.0.1",
			UserAgent: "New Browser/2.0",
		}

		session, err := repo.Create(ctx, params)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if session.Token != params.Token {
			t.Errorf("session.Token = %v, want %v", session.Token, params.Token)
		}
		if session.ActorID != params.ActorID {
			t.Errorf("session.ActorID = %v, want %v", session.ActorID, params.ActorID)
		}
		if session.IPAddress != params.IPAddress {
			t.Errorf("session.IPAddress = %v, want %v", session.IPAddress, params.IPAddress)
		}
		if session.UserAgent != params.UserAgent {
			t.Errorf("session.UserAgent = %v, want %v", session.UserAgent, params.UserAgent)
		}
		if session.SignedInAt.IsZero() {
			t.Error("session.SignedInAt should not be zero")
		}
	})
}

func TestSessionRepository_DeleteByToken(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	// テストデータを作成（User → Profile → Actor → Session）
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("session-delete@example.com").
		Build()
	profileID := testutil.NewProfileBuilder(t, tx).
		WithAtname("sessionuser3").
		Build()
	actorID := testutil.NewActorBuilder(t, tx).
		WithUserID(userID).
		WithProfileID(profileID).
		Build()

	token := "test-session-token-delete"
	_ = testutil.NewSessionBuilder(t, tx).
		WithActorID(actorID).
		WithToken(token).
		Build()

	repo := repository.NewSessionRepository(tx)

	t.Run("セッションを削除できる", func(t *testing.T) {
		// 削除前に存在を確認
		_, err := repo.GetByToken(ctx, token)
		if err != nil {
			t.Fatalf("セッションが存在しません: %v", err)
		}

		// 削除
		err = repo.DeleteByToken(ctx, token)
		if err != nil {
			t.Fatalf("DeleteByToken() error = %v", err)
		}

		// 削除後に存在しないことを確認
		_, err = repo.GetByToken(ctx, token)
		if err != repository.ErrNotFound {
			t.Errorf("GetByToken() after delete should return ErrNotFound, got %v", err)
		}
	})

	t.Run("存在しないトークンの削除はエラーにならない", func(t *testing.T) {
		err := repo.DeleteByToken(ctx, "nonexistent-token-for-delete")
		if err != nil {
			t.Errorf("DeleteByToken() should not return error for non-existent token, got %v", err)
		}
	})
}
