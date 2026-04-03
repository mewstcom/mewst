package usecase_test

import (
	"context"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// mockInserter はテスト用のモック inserter
type mockInserter struct {
	called bool
	args   river.JobArgs
}

func (m *mockInserter) Insert(_ context.Context, args river.JobArgs) (*rivertype.JobInsertResult, error) {
	m.called = true
	m.args = args
	return &rivertype.JobInsertResult{}, nil
}

func TestCreateEmailConfirmationUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	// リポジトリとモックinserterを作成
	emailConfirmRepo := repository.NewEmailConfirmationRepository(tx)
	inserter := &mockInserter{}
	d := dispatcher.NewDispatcher(inserter)

	// ユースケースを実行
	uc := usecase.NewCreateEmailConfirmationUsecase(emailConfirmRepo, d)
	result, err := uc.Execute(ctx, usecase.CreateEmailConfirmationInput{
		Email:  "test@example.com",
		Event:  model.EmailConfirmationEventPasswordReset,
		Locale: "ja",
	})

	// アサーション
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result == nil {
		t.Fatal("Execute() result should not be nil")
	}

	if result.EmailConfirmation == nil {
		t.Fatal("EmailConfirmation should not be nil")
	}

	// メール確認レコードが正しく作成されたか確認
	ec := result.EmailConfirmation
	if ec.Email != "test@example.com" {
		t.Errorf("Email = %v, want %v", ec.Email, "test@example.com")
	}

	if ec.Event != model.EmailConfirmationEventPasswordReset {
		t.Errorf("Event = %v, want %v", ec.Event, model.EmailConfirmationEventPasswordReset)
	}

	// コードが6桁であることを確認
	if len(ec.Code) != 6 {
		t.Errorf("Code length = %v, want 6", len(ec.Code))
	}

	// コードが数字のみであることを確認
	for _, c := range ec.Code {
		if c < '0' || c > '9' {
			t.Errorf("Code contains non-digit character: %c", c)
		}
	}

	// エンキューが呼ばれたことを確認
	if !inserter.called {
		t.Fatal("Insert() が呼ばれていません")
	}

	// SendEmailConfirmationArgs の検証
	emailArgs, ok := inserter.args.(dispatcher.SendEmailConfirmationArgs)
	if !ok {
		t.Fatalf("args の型が SendEmailConfirmationArgs ではありません: %T", inserter.args)
	}
	if emailArgs.Email != "test@example.com" {
		t.Errorf("Email = %s, want test@example.com", emailArgs.Email)
	}
	if emailArgs.Code != ec.Code {
		t.Errorf("Code = %s, want %s", emailArgs.Code, ec.Code)
	}
	if emailArgs.Locale != "ja" {
		t.Errorf("Locale = %s, want ja", emailArgs.Locale)
	}
}

func TestCreateEmailConfirmationUsecase_Execute_EnglishLocale(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	emailConfirmRepo := repository.NewEmailConfirmationRepository(tx)
	inserter := &mockInserter{}
	d := dispatcher.NewDispatcher(inserter)

	uc := usecase.NewCreateEmailConfirmationUsecase(emailConfirmRepo, d)
	result, err := uc.Execute(ctx, usecase.CreateEmailConfirmationInput{
		Email:  "test@example.com",
		Event:  model.EmailConfirmationEventPasswordReset,
		Locale: "en",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result == nil {
		t.Fatal("Execute() result should not be nil")
	}

	// エンキューが呼ばれたことを確認
	if !inserter.called {
		t.Fatal("Insert() が呼ばれていません")
	}

	// 英語ロケールでエンキューされたことを確認
	emailArgs, ok := inserter.args.(dispatcher.SendEmailConfirmationArgs)
	if !ok {
		t.Fatalf("args の型が SendEmailConfirmationArgs ではありません: %T", inserter.args)
	}
	if emailArgs.Locale != "en" {
		t.Errorf("Locale = %s, want en", emailArgs.Locale)
	}
}

func TestCreateEmailConfirmationUsecase_Execute_CodeUniqueness(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	emailConfirmRepo := repository.NewEmailConfirmationRepository(tx)
	inserter := &mockInserter{}
	d := dispatcher.NewDispatcher(inserter)

	uc := usecase.NewCreateEmailConfirmationUsecase(emailConfirmRepo, d)

	// 複数のメール確認を作成してコードが一意であることを確認
	codes := make(map[string]bool)
	for i := 0; i < 10; i++ {
		result, err := uc.Execute(ctx, usecase.CreateEmailConfirmationInput{
			Email:  "test@example.com",
			Event:  model.EmailConfirmationEventPasswordReset,
			Locale: "ja",
		})

		if err != nil {
			t.Fatalf("Execute() error on iteration %d: %v", i, err)
		}

		code := result.EmailConfirmation.Code
		if codes[code] {
			t.Errorf("Code %v is not unique on iteration %d", code, i)
		}
		codes[code] = true
	}
}

func TestCreateEmailConfirmationUsecase_Execute_RecordPersistence(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	emailConfirmRepo := repository.NewEmailConfirmationRepository(tx)
	inserter := &mockInserter{}
	d := dispatcher.NewDispatcher(inserter)

	uc := usecase.NewCreateEmailConfirmationUsecase(emailConfirmRepo, d)
	result, err := uc.Execute(ctx, usecase.CreateEmailConfirmationInput{
		Email:  "persist@example.com",
		Event:  model.EmailConfirmationEventPasswordReset,
		Locale: "ja",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// 作成されたレコードがDBに存在するか確認
	createdEC, err := emailConfirmRepo.GetByID(ctx, result.EmailConfirmation.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if createdEC.Email != "persist@example.com" {
		t.Errorf("createdEC.Email = %v, want %v", createdEC.Email, "persist@example.com")
	}

	if createdEC.Code != result.EmailConfirmation.Code {
		t.Errorf("createdEC.Code = %v, want %v", createdEC.Code, result.EmailConfirmation.Code)
	}
}

func TestCreateEmailConfirmationUsecase_Execute_SignUpEvent(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	emailConfirmRepo := repository.NewEmailConfirmationRepository(tx)
	inserter := &mockInserter{}
	d := dispatcher.NewDispatcher(inserter)

	uc := usecase.NewCreateEmailConfirmationUsecase(emailConfirmRepo, d)
	result, err := uc.Execute(ctx, usecase.CreateEmailConfirmationInput{
		Email:  "signup@example.com",
		Event:  model.EmailConfirmationEventSignUp,
		Locale: "ja",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// イベントタイプが正しく設定されているか確認
	if result.EmailConfirmation.Event != model.EmailConfirmationEventSignUp {
		t.Errorf("Event = %v, want %v", result.EmailConfirmation.Event, model.EmailConfirmationEventSignUp)
	}
}
