package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

func TestGetSucceededEmailConfirmationUsecase_Execute_Success(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	// 確認済みのメール確認レコードを作成
	now := time.Now()
	emailConfirmationID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithEvent("password_reset").
		WithCode("123456").
		WithSucceededAt(now).
		Build()

	// ユースケースを実行
	emailConfirmRepo := repository.NewEmailConfirmationRepository(tx)
	uc := usecase.NewGetSucceededEmailConfirmationUsecase(emailConfirmRepo)
	result, err := uc.Execute(ctx, usecase.GetSucceededEmailConfirmationInput{
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

	if result.EmailConfirmation.SucceededAt == nil {
		t.Error("SucceededAt should not be nil for succeeded email confirmation")
	}
}

func TestGetSucceededEmailConfirmationUsecase_Execute_NotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	// 存在しないIDで実行
	emailConfirmRepo := repository.NewEmailConfirmationRepository(tx)
	uc := usecase.NewGetSucceededEmailConfirmationUsecase(emailConfirmRepo)
	_, err := uc.Execute(ctx, usecase.GetSucceededEmailConfirmationInput{
		ID: uuid.New(),
	})

	if err == nil {
		t.Fatal("Execute() should return error for non-existent ID")
	}

	if !errors.Is(err, usecase.ErrNotFound) {
		t.Errorf("Execute() error should be ErrNotFound, got %v", err)
	}
}

func TestGetSucceededEmailConfirmationUsecase_Execute_NotSucceeded(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	// 未確認のメール確認レコードを作成（succeeded_atがNULL）
	emailConfirmationID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithEvent("password_reset").
		WithCode("123456").
		Build()

	emailConfirmRepo := repository.NewEmailConfirmationRepository(tx)
	uc := usecase.NewGetSucceededEmailConfirmationUsecase(emailConfirmRepo)
	_, err := uc.Execute(ctx, usecase.GetSucceededEmailConfirmationInput{
		ID: emailConfirmationID,
	})

	if err == nil {
		t.Fatal("Execute() should return error for unconfirmed email confirmation")
	}

	if !errors.Is(err, usecase.ErrNotFound) {
		t.Errorf("Execute() error should be ErrNotFound, got %v", err)
	}
}
