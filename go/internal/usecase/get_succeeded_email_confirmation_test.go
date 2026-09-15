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

func TestGetSucceededEmailConfirmationUsecase_Execute_Success(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	// 確認済みのメール確認レコードを作成
	now := time.Now()
	emailConfirmationID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("usecase-succeeded-email-confirmation@example.com").
		WithEvent("password_reset").
		WithCode("123456").
		WithSucceededAt(now).
		Build()

	// ユースケースを実行
	emailConfirmRepo := repository.NewEmailConfirmationRepository(testutil.QueriesWithTx(tx))
	uc := usecase.NewGetSucceededEmailConfirmationUsecase(emailConfirmRepo)
	result, err := uc.Execute(ctx, usecase.GetSucceededEmailConfirmationInput{
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

	if result.EmailConfirmation.Email != "usecase-succeeded-email-confirmation@example.com" {
		t.Errorf("Email = %v、期待値 = %v", result.EmailConfirmation.Email, "usecase-succeeded-email-confirmation@example.com")
	}

	if result.EmailConfirmation.SucceededAt == nil {
		t.Error("確認済みのメール確認のSucceededAt = nil、非nilを期待")
	}
}

func TestGetSucceededEmailConfirmationUsecase_Execute_NotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	// 存在しないIDで実行
	emailConfirmRepo := repository.NewEmailConfirmationRepository(testutil.QueriesWithTx(tx))
	uc := usecase.NewGetSucceededEmailConfirmationUsecase(emailConfirmRepo)
	_, err := uc.Execute(ctx, usecase.GetSucceededEmailConfirmationInput{
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

func TestGetSucceededEmailConfirmationUsecase_Execute_NotSucceeded(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	// 未確認のメール確認レコードを作成 (succeeded_atがNULL)
	emailConfirmationID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("usecase-succeeded-email-confirmation-not-succeeded@example.com").
		WithEvent("password_reset").
		WithCode("123456").
		Build()

	emailConfirmRepo := repository.NewEmailConfirmationRepository(testutil.QueriesWithTx(tx))
	uc := usecase.NewGetSucceededEmailConfirmationUsecase(emailConfirmRepo)
	_, err := uc.Execute(ctx, usecase.GetSucceededEmailConfirmationInput{
		ID: emailConfirmationID,
	})

	if err == nil {
		t.Fatal("未確認のメール確認でExecute()がエラーを返すことを期待したが、nilだった")
	}

	ae := model.AsAppError(err)
	if ae == nil {
		t.Fatalf("Execute()のエラー = %v、*model.AppErrorを期待", err)
	}
	if ae.Code != model.AppErrCodeResourceNotFound {
		t.Errorf("Execute()のエラーコード = %d、期待値 = %d", ae.Code, model.AppErrCodeResourceNotFound)
	}
}
