package repository_test

import (
	"context"
	"testing"

	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

func TestSessionRepository_FindByToken(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	// テストデータを作成 (User → Profile → Actor → Session)
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

	repo := repository.NewSessionRepository(testutil.QueriesWithTx(tx))

	t.Run("存在するセッションをトークンで取得できる", func(t *testing.T) {
		session, err := repo.FindByToken(ctx, token)
		if err != nil {
			t.Fatalf("FindByToken() error = %v", err)
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

	t.Run("存在しないトークンはnilを返す", func(t *testing.T) {
		session, err := repo.FindByToken(ctx, "nonexistent-token")
		if err != nil {
			t.Errorf("FindByToken() error = %v, want nil", err)
		}
		if session != nil {
			t.Errorf("FindByToken() session = %v, want nil", session)
		}
	})
}

func TestSessionRepository_FindAuthByToken(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	// Set up user / profile / actor / session for the happy path.
	// [Ja] 正常系で参照する user / profile / actor / session を準備する。
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("session-findauth@example.com").
		Build()
	profileID := testutil.NewProfileBuilder(t, tx).
		WithAtname("authlookupuser").
		Build()
	actorID := testutil.NewActorBuilder(t, tx).
		WithUserID(userID).
		WithProfileID(profileID).
		Build()

	token := "test-session-token-findauth"
	_ = testutil.NewSessionBuilder(t, tx).
		WithActorID(actorID).
		WithToken(token).
		Build()

	repo := repository.NewSessionRepository(testutil.QueriesWithTx(tx))

	t.Run("既存のトークンで user / actor / profile を 1 度に取得できる", func(t *testing.T) {
		lookup, err := repo.FindAuthByToken(ctx, token)
		if err != nil {
			t.Fatalf("FindAuthByToken() error = %v", err)
		}
		if lookup == nil {
			t.Fatal("FindAuthByToken() = nil, want AuthLookup")
		}
		if lookup.Actor == nil || lookup.Actor.ID != actorID {
			t.Errorf("lookup.Actor.ID = %v, want %v", lookup.Actor, actorID)
		}
		if lookup.User == nil || lookup.User.ID != userID {
			t.Errorf("lookup.User.ID = %v, want %v", lookup.User, userID)
		}
		if lookup.User != nil && lookup.User.Email != "session-findauth@example.com" {
			t.Errorf("lookup.User.Email = %q, want %q", lookup.User.Email, "session-findauth@example.com")
		}
		if lookup.Profile == nil || lookup.Profile.ID != profileID {
			t.Errorf("lookup.Profile.ID = %v, want %v", lookup.Profile, profileID)
		}
		if lookup.Profile != nil && lookup.Profile.Atname != "authlookupuser" {
			t.Errorf("lookup.Profile.Atname = %q, want %q", lookup.Profile.Atname, "authlookupuser")
		}
	})

	t.Run("存在しないトークンは (nil, nil) を返す", func(t *testing.T) {
		lookup, err := repo.FindAuthByToken(ctx, "nonexistent-token-for-findauth")
		if err != nil {
			t.Errorf("FindAuthByToken() error = %v, want nil", err)
		}
		if lookup != nil {
			t.Errorf("FindAuthByToken() = %v, want nil", lookup)
		}
	})
}

func TestSessionRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	// テストデータを作成 (User → Profile → Actor)
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

	repo := repository.NewSessionRepository(testutil.QueriesWithTx(tx))

	t.Run("セッションを作成できる", func(t *testing.T) {
		params := repository.CreateSessionInput{
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

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	// テストデータを作成 (User → Profile → Actor → Session)
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

	repo := repository.NewSessionRepository(testutil.QueriesWithTx(tx))

	t.Run("セッションを削除できる", func(t *testing.T) {
		// 削除前に存在を確認
		_, err := repo.FindByToken(ctx, token)
		if err != nil {
			t.Fatalf("セッションが存在しません: %v", err)
		}

		// 削除
		err = repo.DeleteByToken(ctx, token)
		if err != nil {
			t.Fatalf("DeleteByToken() error = %v", err)
		}

		// 削除後に存在しないことを確認
		session, err := repo.FindByToken(ctx, token)
		if err != nil {
			t.Errorf("FindByToken() after delete error = %v, want nil", err)
		}
		if session != nil {
			t.Errorf("FindByToken() after delete session = %v, want nil", session)
		}
	})

	t.Run("存在しないトークンの削除はエラーにならない", func(t *testing.T) {
		err := repo.DeleteByToken(ctx, "nonexistent-token-for-delete")
		if err != nil {
			t.Errorf("DeleteByToken() should not return error for non-existent token, got %v", err)
		}
	})
}
