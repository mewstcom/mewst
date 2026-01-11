package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/internal/repository"
	"github.com/mewstcom/mewst/internal/testutil"
	"github.com/mewstcom/mewst/internal/usecase"
)

func TestConfirmEmailUsecase_Execute_Success(t *testing.T) {
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
	emailConfirmRepo := repository.NewEmailConfirmationRepository(tx)
	uc := usecase.NewConfirmEmailUsecase(emailConfirmRepo)
	result, err := uc.Execute(ctx, usecase.ConfirmEmailInput{
		EmailConfirmationID: emailConfirmationID,
		Code:                "123456",
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

	// succeeded_at が設定されていることを確認
	if result.EmailConfirmation.SucceededAt == nil {
		t.Error("SucceededAt should not be nil after confirmation")
	}

	// 他のフィールドが正しいことを確認
	if result.EmailConfirmation.Email != "test@example.com" {
		t.Errorf("Email = %v, want %v", result.EmailConfirmation.Email, "test@example.com")
	}

	if result.EmailConfirmation.Code != "123456" {
		t.Errorf("Code = %v, want %v", result.EmailConfirmation.Code, "123456")
	}
}

func TestConfirmEmailUsecase_Execute_NotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	// 存在しないIDで実行
	emailConfirmRepo := repository.NewEmailConfirmationRepository(tx)
	uc := usecase.NewConfirmEmailUsecase(emailConfirmRepo)
	_, err := uc.Execute(ctx, usecase.ConfirmEmailInput{
		EmailConfirmationID: uuid.New(),
		Code:                "123456",
	})

	// ErrNotFoundが返されることを確認
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("Execute() error = %v, want %v", err, repository.ErrNotFound)
	}
}

func TestConfirmEmailUsecase_Execute_CodeMismatch(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	// テスト用メール確認レコードを作成
	emailConfirmationID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithEvent("password_reset").
		WithCode("123456").
		Build()

	// 間違ったコードで実行
	emailConfirmRepo := repository.NewEmailConfirmationRepository(tx)
	uc := usecase.NewConfirmEmailUsecase(emailConfirmRepo)
	_, err := uc.Execute(ctx, usecase.ConfirmEmailInput{
		EmailConfirmationID: emailConfirmationID,
		Code:                "999999",
	})

	// ErrCodeMismatchが返されることを確認
	if !errors.Is(err, usecase.ErrCodeMismatch) {
		t.Errorf("Execute() error = %v, want %v", err, usecase.ErrCodeMismatch)
	}
}

func TestConfirmEmailUsecase_Execute_Expired(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	// 有効期限切れのメール確認レコードを作成（16分前）
	expiredTime := time.Now().Add(-16 * time.Minute)
	emailConfirmationID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithEvent("password_reset").
		WithCode("123456").
		WithCreatedAt(expiredTime).
		Build()

	// 実行
	emailConfirmRepo := repository.NewEmailConfirmationRepository(tx)
	uc := usecase.NewConfirmEmailUsecase(emailConfirmRepo)
	_, err := uc.Execute(ctx, usecase.ConfirmEmailInput{
		EmailConfirmationID: emailConfirmationID,
		Code:                "123456",
	})

	// ErrNotFoundが返されることを確認（期限切れはGetActiveByIDでフィルタされる）
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("Execute() error = %v, want %v", err, repository.ErrNotFound)
	}
}

func TestConfirmEmailUsecase_Execute_AlreadySucceeded(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	// 既に確認済みのメール確認レコードを作成
	succeededAt := time.Now().Add(-1 * time.Hour)
	emailConfirmationID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithEvent("password_reset").
		WithCode("123456").
		WithSucceededAt(succeededAt).
		Build()

	// 実行
	emailConfirmRepo := repository.NewEmailConfirmationRepository(tx)
	uc := usecase.NewConfirmEmailUsecase(emailConfirmRepo)
	_, err := uc.Execute(ctx, usecase.ConfirmEmailInput{
		EmailConfirmationID: emailConfirmationID,
		Code:                "123456",
	})

	// ErrNotFoundが返されることを確認（確認済みはGetActiveByIDでフィルタされる）
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("Execute() error = %v, want %v", err, repository.ErrNotFound)
	}
}

func TestConfirmEmailUsecase_Execute_WithinExpiry(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	// 有効期限内のメール確認レコードを作成（14分前 = まだ有効）
	validTime := time.Now().Add(-14 * time.Minute)
	emailConfirmationID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithEvent("password_reset").
		WithCode("123456").
		WithCreatedAt(validTime).
		Build()

	// 実行
	emailConfirmRepo := repository.NewEmailConfirmationRepository(tx)
	uc := usecase.NewConfirmEmailUsecase(emailConfirmRepo)
	result, err := uc.Execute(ctx, usecase.ConfirmEmailInput{
		EmailConfirmationID: emailConfirmationID,
		Code:                "123456",
	})

	// 成功することを確認
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.EmailConfirmation.SucceededAt == nil {
		t.Error("SucceededAt should not be nil after confirmation")
	}
}
