package usecase_test

import (
	"context"
	"testing"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/validator"
)

func TestCreateSignInUsecase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("正常系: サインインしてセッションを作成できる", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		ctx := context.Background()

		// テストデータを作成
		testEmail := "signin-uc-success@example.com"
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail(testEmail).
			Build()

		profileID := testutil.NewProfileBuilder(t, tx).
			WithAtname("signinuc").
			Build()

		testutil.NewActorBuilder(t, tx).
			WithUserID(userID).
			WithProfileID(profileID).
			Build()

		// リポジトリを作成
		userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
		actorRepo := repository.NewActorRepository(testutil.QueriesWithTx(tx))
		sessionRepo := repository.NewSessionRepository(testutil.QueriesWithTx(tx))

		signInValidator := validator.NewSignInCreateValidator(userRepo)
		uc := usecase.NewCreateSignInUsecase(signInValidator, actorRepo, sessionRepo)

		output, err := uc.Execute(ctx, usecase.CreateSignInInput{
			Email:     testEmail,
			Password:  "password", // ビルダーのデフォルトパスワード
			IPAddress: "192.168.1.1",
			UserAgent: "Mozilla/5.0",
		})

		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("Execute() returned nil output")
		}
		if output.Token == "" {
			t.Error("Execute() returned empty token")
		}
		if output.Session == nil {
			t.Fatal("Execute() returned nil session")
		}
	})

	t.Run("正常系: 各呼び出しで異なるトークンが生成される", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		ctx := context.Background()

		testEmail := "signin-uc-unique@example.com"
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail(testEmail).
			Build()

		profileID := testutil.NewProfileBuilder(t, tx).
			WithAtname("signinucuniq").
			Build()

		testutil.NewActorBuilder(t, tx).
			WithUserID(userID).
			WithProfileID(profileID).
			Build()

		userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
		actorRepo := repository.NewActorRepository(testutil.QueriesWithTx(tx))
		sessionRepo := repository.NewSessionRepository(testutil.QueriesWithTx(tx))

		signInValidator := validator.NewSignInCreateValidator(userRepo)
		uc := usecase.NewCreateSignInUsecase(signInValidator, actorRepo, sessionRepo)

		input := usecase.CreateSignInInput{
			Email:     testEmail,
			Password:  "password",
			IPAddress: "192.168.1.1",
			UserAgent: "TestAgent",
		}

		output1, err := uc.Execute(ctx, input)
		if err != nil {
			t.Fatalf("Execute() first call error = %v", err)
		}

		output2, err := uc.Execute(ctx, input)
		if err != nil {
			t.Fatalf("Execute() second call error = %v", err)
		}

		if output1.Token == output2.Token {
			t.Error("Execute() returned same token for different calls")
		}
	})

	t.Run("異常系: バリデーションエラー", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		ctx := context.Background()

		userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
		actorRepo := repository.NewActorRepository(testutil.QueriesWithTx(tx))
		sessionRepo := repository.NewSessionRepository(testutil.QueriesWithTx(tx))

		signInValidator := validator.NewSignInCreateValidator(userRepo)
		uc := usecase.NewCreateSignInUsecase(signInValidator, actorRepo, sessionRepo)

		output, err := uc.Execute(ctx, usecase.CreateSignInInput{
			Email:     "",
			Password:  "",
			IPAddress: "192.168.1.1",
			UserAgent: "Mozilla/5.0",
		})

		if output != nil {
			t.Error("expected nil output")
		}
		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatal("expected ValidationError, got nil")
		}
		if !ve.HasFieldError("email") {
			t.Error("expected email field error")
		}
	})

	t.Run("異常系: ユーザーが見つからない場合", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		ctx := context.Background()

		userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
		actorRepo := repository.NewActorRepository(testutil.QueriesWithTx(tx))
		sessionRepo := repository.NewSessionRepository(testutil.QueriesWithTx(tx))

		signInValidator := validator.NewSignInCreateValidator(userRepo)
		uc := usecase.NewCreateSignInUsecase(signInValidator, actorRepo, sessionRepo)

		output, err := uc.Execute(ctx, usecase.CreateSignInInput{
			Email:     "nonexistent@example.com",
			Password:  "password123",
			IPAddress: "192.168.1.1",
			UserAgent: "Mozilla/5.0",
		})

		if output != nil {
			t.Error("expected nil output")
		}
		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatal("expected ValidationError, got nil")
		}
		if len(ve.Global) == 0 {
			t.Error("expected global error")
		}
	})

	t.Run("異常系: パスワードが正しくない場合", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		ctx := context.Background()

		testEmail := "signin-uc-wrongpw@example.com"
		testutil.NewUserBuilder(t, tx).
			WithEmail(testEmail).
			Build()

		userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
		actorRepo := repository.NewActorRepository(testutil.QueriesWithTx(tx))
		sessionRepo := repository.NewSessionRepository(testutil.QueriesWithTx(tx))

		signInValidator := validator.NewSignInCreateValidator(userRepo)
		uc := usecase.NewCreateSignInUsecase(signInValidator, actorRepo, sessionRepo)

		output, err := uc.Execute(ctx, usecase.CreateSignInInput{
			Email:     testEmail,
			Password:  "wrongpassword",
			IPAddress: "192.168.1.1",
			UserAgent: "Mozilla/5.0",
		})

		if output != nil {
			t.Error("expected nil output")
		}
		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatal("expected ValidationError, got nil")
		}
		if len(ve.Global) == 0 {
			t.Error("expected global error")
		}
	})
}
