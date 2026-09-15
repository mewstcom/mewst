package usecase_test

import (
	"context"
	"testing"

	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

func TestCreateSessionUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	// テストデータを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("usecase-create-session@example.com").
		Build()

	profileID := testutil.NewProfileBuilder(t, tx).
		WithAtname("testuser").
		Build()

	actorID := testutil.NewActorBuilder(t, tx).
		WithUserID(userID).
		WithProfileID(profileID).
		Build()

	// リポジトリを作成 (トランザクションを使用)
	actorRepo := repository.NewActorRepository(testutil.QueriesWithTx(tx))
	sessionRepo := repository.NewSessionRepository(testutil.QueriesWithTx(tx))

	// ユースケースを実行
	uc := usecase.NewCreateSessionUsecase(actorRepo, sessionRepo)
	result, err := uc.Execute(ctx, usecase.CreateSessionInput{
		UserID:    userID,
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0 (Test)",
	})

	// アサーション
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}

	if result == nil {
		t.Fatal("Execute()の結果 = nil、非nilを期待")
	}

	if result.Token == "" {
		t.Error("Token = 空文字列、非空を期待")
	}

	if result.Session == nil {
		t.Fatal("Session = nil、非nilを期待")
	}

	if result.Session.ActorID != actorID {
		t.Errorf("Session.ActorID = %v、期待値 = %v", result.Session.ActorID, actorID)
	}

	if result.Session.Token != result.Token {
		t.Errorf("Session.Token = %v、期待値 = %v", result.Session.Token, result.Token)
	}

	if result.Session.IPAddress != "192.168.1.1" {
		t.Errorf("Session.IPAddress = %v、期待値 = %v", result.Session.IPAddress, "192.168.1.1")
	}

	if result.Session.UserAgent != "Mozilla/5.0 (Test)" {
		t.Errorf("Session.UserAgent = %v、期待値 = %v", result.Session.UserAgent, "Mozilla/5.0 (Test)")
	}

	// 作成されたセッションがDBに存在するか確認
	createdSession, err := sessionRepo.FindByToken(ctx, result.Token)
	if err != nil {
		t.Fatalf("FindByToken()のエラー = %v", err)
	}

	if createdSession.ID != result.Session.ID {
		t.Errorf("createdSession.ID = %v、期待値 = %v", createdSession.ID, result.Session.ID)
	}
}

func TestCreateSessionUsecase_Execute_EmptyIPAddress(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	// テストデータを作成
	userID := testutil.NewUserBuilder(t, tx).Build()
	profileID := testutil.NewProfileBuilder(t, tx).Build()
	testutil.NewActorBuilder(t, tx).
		WithUserID(userID).
		WithProfileID(profileID).
		Build()

	actorRepo := repository.NewActorRepository(testutil.QueriesWithTx(tx))
	sessionRepo := repository.NewSessionRepository(testutil.QueriesWithTx(tx))
	uc := usecase.NewCreateSessionUsecase(actorRepo, sessionRepo)

	// IPアドレスとUser-Agentが空の場合でも正常に動作することを確認
	result, err := uc.Execute(ctx, usecase.CreateSessionInput{
		UserID:    userID,
		IPAddress: "",
		UserAgent: "",
	})

	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}

	if result.Session.IPAddress != "" {
		t.Errorf("Session.IPAddress = %v、空文字列を期待", result.Session.IPAddress)
	}

	if result.Session.UserAgent != "" {
		t.Errorf("Session.UserAgent = %v、空文字列を期待", result.Session.UserAgent)
	}
}

func TestCreateSessionUsecase_Execute_TokenUniqueness(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	// テストデータを作成
	userID := testutil.NewUserBuilder(t, tx).Build()
	profileID := testutil.NewProfileBuilder(t, tx).Build()
	testutil.NewActorBuilder(t, tx).
		WithUserID(userID).
		WithProfileID(profileID).
		Build()

	actorRepo := repository.NewActorRepository(testutil.QueriesWithTx(tx))
	sessionRepo := repository.NewSessionRepository(testutil.QueriesWithTx(tx))
	uc := usecase.NewCreateSessionUsecase(actorRepo, sessionRepo)

	// 複数のセッションを作成してトークンが一意であることを確認
	tokens := make(map[string]bool)
	for i := 0; i < 10; i++ {
		result, err := uc.Execute(ctx, usecase.CreateSessionInput{
			UserID:    userID,
			IPAddress: "127.0.0.1",
			UserAgent: "Test",
		})

		if err != nil {
			t.Fatalf("%d回目のExecute()のエラー = %v", i, err)
		}

		if tokens[result.Token] {
			t.Errorf("Token %vが%d回目の繰り返しで重複した", result.Token, i)
		}
		tokens[result.Token] = true
	}
}
