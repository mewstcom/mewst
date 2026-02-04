package usecase_test

import (
	"context"
	"testing"

	"github.com/mewstcom/mewst/go/internal/email"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

func TestCreateEmailConfirmationUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	// リポジトリとモックメール送信者を作成
	emailConfirmRepo := repository.NewEmailConfirmationRepository(tx)
	emailSender := email.NewNoopSender()

	// ユースケースを実行
	uc := usecase.NewCreateEmailConfirmationUsecase(emailConfirmRepo, emailSender)
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

	// メールが送信されたことを確認
	if len(emailSender.SentEmails) != 1 {
		t.Fatalf("SentEmails count = %v, want 1", len(emailSender.SentEmails))
	}

	sentEmail := emailSender.SentEmails[0]
	if sentEmail.To != "test@example.com" {
		t.Errorf("SentEmail.To = %v, want %v", sentEmail.To, "test@example.com")
	}

	if sentEmail.Subject != "[Mewst] 確認用コード" {
		t.Errorf("SentEmail.Subject = %v, want %v", sentEmail.Subject, "[Mewst] 確認用コード")
	}
}

func TestCreateEmailConfirmationUsecase_Execute_EnglishLocale(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	emailConfirmRepo := repository.NewEmailConfirmationRepository(tx)
	emailSender := email.NewNoopSender()

	uc := usecase.NewCreateEmailConfirmationUsecase(emailConfirmRepo, emailSender)
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

	// 英語の件名でメールが送信されたことを確認
	if len(emailSender.SentEmails) != 1 {
		t.Fatalf("SentEmails count = %v, want 1", len(emailSender.SentEmails))
	}

	sentEmail := emailSender.SentEmails[0]
	if sentEmail.Subject != "[Mewst] Confirmation code" {
		t.Errorf("SentEmail.Subject = %v, want %v", sentEmail.Subject, "[Mewst] Confirmation code")
	}
}

func TestCreateEmailConfirmationUsecase_Execute_CodeUniqueness(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	emailConfirmRepo := repository.NewEmailConfirmationRepository(tx)
	emailSender := email.NewNoopSender()

	uc := usecase.NewCreateEmailConfirmationUsecase(emailConfirmRepo, emailSender)

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
	emailSender := email.NewNoopSender()

	uc := usecase.NewCreateEmailConfirmationUsecase(emailConfirmRepo, emailSender)
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
	emailSender := email.NewNoopSender()

	uc := usecase.NewCreateEmailConfirmationUsecase(emailConfirmRepo, emailSender)
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
