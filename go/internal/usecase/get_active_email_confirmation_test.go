package usecase_test

import (
	"context"
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

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	// テスト用メール確認レコードを作成
	emailConfirmationID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("usecase-active-email-confirmation@example.com").
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
		t.Fatalf("Execute()のエラー = %v", err)
	}

	if result == nil {
		t.Fatal("Execute()の結果 = nil、非nilを期待")
	}

	if result.EmailConfirmation == nil {
		t.Fatal("EmailConfirmation = nil、非nilを期待")
	}

	if result.EmailConfirmation.Email != "usecase-active-email-confirmation@example.com" {
		t.Errorf("Email = %v、期待値 = %v", result.EmailConfirmation.Email, "usecase-active-email-confirmation@example.com")
	}

	if result.EmailConfirmation.SucceededAt != nil {
		t.Error("有効なメール確認のSucceededAt = 非nil、nilを期待")
	}
}

func TestGetActiveEmailConfirmationUsecase_Execute_NotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	// 存在しないIDで実行
	emailConfirmRepo := repository.NewEmailConfirmationRepository(testutil.QueriesWithTx(tx))
	uc := usecase.NewGetActiveEmailConfirmationUsecase(emailConfirmRepo)
	_, err := uc.Execute(ctx, usecase.GetActiveEmailConfirmationInput{
		ID: model.EmailConfirmationID(uuid.New()),
	})

	if err == nil {
		t.Fatal("存在しないIDでExecute()がエラーを返すことを期待したが、nilだった")
	}

	ae := model.AsAppError(err)
	if ae == nil {
		t.Fatalf("Execute()のエラー = %v、*model.AppErrorを期待", err)
	}
	if ae.Code != model.AppErrCodeResourceNotFound {
		t.Errorf("Execute()のエラーコード = %d、期待値 = %d", ae.Code, model.AppErrCodeResourceNotFound)
	}
}

func TestGetActiveEmailConfirmationUsecase_Execute_Expired(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	// 期限切れのメール確認レコードを作成 (16分前)
	expiredTime := time.Now().Add(-16 * time.Minute)
	emailConfirmationID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("usecase-active-email-confirmation-expired@example.com").
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
		t.Fatal("期限切れのメール確認でExecute()がエラーを返すことを期待したが、nilだった")
	}

	ae := model.AsAppError(err)
	if ae == nil {
		t.Fatalf("Execute()のエラー = %v、*model.AppErrorを期待", err)
	}
	if ae.Code != model.AppErrCodeResourceNotFound {
		t.Errorf("Execute()のエラーコード = %d、期待値 = %d", ae.Code, model.AppErrCodeResourceNotFound)
	}
}
