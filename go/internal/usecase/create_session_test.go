package usecase_test

import (
	"context"
	"testing"

	"github.com/mewstcom/mewst/internal/repository"
	"github.com/mewstcom/mewst/internal/testutil"
	"github.com/mewstcom/mewst/internal/usecase"
)

func TestCreateSessionUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	// テストデータを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("test@example.com").
		Build()

	profileID := testutil.NewProfileBuilder(t, tx).
		WithAtname("testuser").
		Build()

	actorID := testutil.NewActorBuilder(t, tx).
		WithUserID(userID).
		WithProfileID(profileID).
		Build()

	// リポジトリを作成（トランザクションを使用）
	sessionRepo := repository.NewSessionRepository(tx)

	// ユースケースを実行
	uc := usecase.NewCreateSessionUsecase(sessionRepo)
	result, err := uc.Execute(ctx, usecase.CreateSessionInput{
		ActorID:   actorID,
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0 (Test)",
	})

	// アサーション
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result == nil {
		t.Fatal("Execute() result should not be nil")
	}

	if result.Token == "" {
		t.Error("Token should not be empty")
	}

	if result.Session == nil {
		t.Fatal("Session should not be nil")
	}

	if result.Session.ActorID != actorID {
		t.Errorf("Session.ActorID = %v, want %v", result.Session.ActorID, actorID)
	}

	if result.Session.Token != result.Token {
		t.Errorf("Session.Token = %v, want %v", result.Session.Token, result.Token)
	}

	if result.Session.IPAddress != "192.168.1.1" {
		t.Errorf("Session.IPAddress = %v, want %v", result.Session.IPAddress, "192.168.1.1")
	}

	if result.Session.UserAgent != "Mozilla/5.0 (Test)" {
		t.Errorf("Session.UserAgent = %v, want %v", result.Session.UserAgent, "Mozilla/5.0 (Test)")
	}

	// 作成されたセッションがDBに存在するか確認
	createdSession, err := sessionRepo.GetByToken(ctx, result.Token)
	if err != nil {
		t.Fatalf("GetByToken() error = %v", err)
	}

	if createdSession.ID != result.Session.ID {
		t.Errorf("createdSession.ID = %v, want %v", createdSession.ID, result.Session.ID)
	}
}

func TestCreateSessionUsecase_Execute_EmptyIPAddress(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	// テストデータを作成
	userID := testutil.NewUserBuilder(t, tx).Build()
	profileID := testutil.NewProfileBuilder(t, tx).Build()
	actorID := testutil.NewActorBuilder(t, tx).
		WithUserID(userID).
		WithProfileID(profileID).
		Build()

	sessionRepo := repository.NewSessionRepository(tx)
	uc := usecase.NewCreateSessionUsecase(sessionRepo)

	// IPアドレスとUser-Agentが空の場合でも正常に動作することを確認
	result, err := uc.Execute(ctx, usecase.CreateSessionInput{
		ActorID:   actorID,
		IPAddress: "",
		UserAgent: "",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.Session.IPAddress != "" {
		t.Errorf("Session.IPAddress = %v, want empty string", result.Session.IPAddress)
	}

	if result.Session.UserAgent != "" {
		t.Errorf("Session.UserAgent = %v, want empty string", result.Session.UserAgent)
	}
}

func TestCreateSessionUsecase_Execute_TokenUniqueness(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	// テストデータを作成
	userID := testutil.NewUserBuilder(t, tx).Build()
	profileID := testutil.NewProfileBuilder(t, tx).Build()
	actorID := testutil.NewActorBuilder(t, tx).
		WithUserID(userID).
		WithProfileID(profileID).
		Build()

	sessionRepo := repository.NewSessionRepository(tx)
	uc := usecase.NewCreateSessionUsecase(sessionRepo)

	// 複数のセッションを作成してトークンが一意であることを確認
	tokens := make(map[string]bool)
	for i := 0; i < 10; i++ {
		result, err := uc.Execute(ctx, usecase.CreateSessionInput{
			ActorID:   actorID,
			IPAddress: "127.0.0.1",
			UserAgent: "Test",
		})

		if err != nil {
			t.Fatalf("Execute() error on iteration %d: %v", i, err)
		}

		if tokens[result.Token] {
			t.Errorf("Token %v is not unique on iteration %d", result.Token, i)
		}
		tokens[result.Token] = true
	}
}
