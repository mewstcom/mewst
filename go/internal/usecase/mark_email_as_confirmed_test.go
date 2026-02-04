package usecase_test

import (
	"context"
	"testing"

	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

func TestMarkEmailAsConfirmedUsecase_Execute_Success(t *testing.T) {
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
	uc := usecase.NewMarkEmailAsConfirmedUsecase(emailConfirmRepo)
	result, err := uc.Execute(ctx, usecase.MarkEmailAsConfirmedInput{
		EmailConfirmationID: emailConfirmationID,
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
}
