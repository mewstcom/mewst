package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

func TestGetActiveEmailConfirmationUsecase_Execute_Success(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	// テスト用メール確認レコードを作成
	emailConfirmationID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithEvent("password_reset").
		WithCode("123456").
		Build()

	// ユースケースを実行
	emailConfirmRepo := repository.NewEmailConfirmationRepository(testutil.QueriesWithTx(tx))
	uc := usecase.NewGetActiveEmailConfirmationUsecase(emailConfirmRepo)
	result, err := uc.Execute(ctx, usecase.GetActiveEmailConfirmationInput{
		ID: emailConfirmationID,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result == nil {
		t.Fatal("Execute() result should not be nil")
	}

	if result.EmailConfirmation == nil {
		t.Fatal("EmailConfirmation should not be nil")
	}

	if result.EmailConfirmation.Email != "test@example.com" {
		t.Errorf("Email = %v, want %v", result.EmailConfirmation.Email, "test@example.com")
	}

	if result.EmailConfirmation.SucceededAt != nil {
		t.Error("SucceededAt should be nil for active email confirmation")
	}
}

func TestGetActiveEmailConfirmationUsecase_Execute_NotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	// 存在しないIDで実行
	emailConfirmRepo := repository.NewEmailConfirmationRepository(testutil.QueriesWithTx(tx))
	uc := usecase.NewGetActiveEmailConfirmationUsecase(emailConfirmRepo)
	_, err := uc.Execute(ctx, usecase.GetActiveEmailConfirmationInput{
		ID: model.EmailConfirmationID(uuid.New()),
	})

	if err == nil {
		t.Fatal("Execute() should return error for non-existent ID")
	}

	if !errors.Is(err, usecase.ErrNotFound) {
		t.Errorf("Execute() error should be ErrNotFound, got %v", err)
	}
}

func TestGetActiveEmailConfirmationUsecase_Execute_Expired(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	// 期限切れのメール確認レコードを作成（16分前）
	expiredTime := time.Now().Add(-16 * time.Minute)
	emailConfirmationID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithEvent("password_reset").
		WithCode("123456").
		WithCreatedAt(expiredTime).
		Build()

	emailConfirmRepo := repository.NewEmailConfirmationRepository(testutil.QueriesWithTx(tx))
	uc := usecase.NewGetActiveEmailConfirmationUsecase(emailConfirmRepo)
	_, err := uc.Execute(ctx, usecase.GetActiveEmailConfirmationInput{
		ID: emailConfirmationID,
	})

	if err == nil {
		t.Fatal("Execute() should return error for expired email confirmation")
	}

	if !errors.Is(err, usecase.ErrNotFound) {
		t.Errorf("Execute() error should be ErrNotFound, got %v", err)
	}
}
