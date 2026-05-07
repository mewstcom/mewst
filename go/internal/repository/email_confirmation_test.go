package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

func TestEmailConfirmationRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	repo := repository.NewEmailConfirmationRepository(testutil.QueriesWithTx(tx))

	t.Run("メール確認を作成できる", func(t *testing.T) {
		params := repository.CreateEmailConfirmationInput{
			Email: "create-test@example.com",
			Event: model.EmailConfirmationEventPasswordReset,
			Code:  "123456",
		}

		ec, err := repo.Create(ctx, params)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if ec.Email != params.Email {
			t.Errorf("ec.Email = %v, want %v", ec.Email, params.Email)
		}
		if ec.Event != params.Event {
			t.Errorf("ec.Event = %v, want %v", ec.Event, params.Event)
		}
		if ec.Code != params.Code {
			t.Errorf("ec.Code = %v, want %v", ec.Code, params.Code)
		}
		if ec.SucceededAt != nil {
			t.Errorf("ec.SucceededAt should be nil, got %v", ec.SucceededAt)
		}
		if ec.CreatedAt.IsZero() {
			t.Error("ec.CreatedAt should not be zero")
		}
	})
}

func TestEmailConfirmationRepository_FindByID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	email := "getbyid-test@example.com"
	id := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail(email).
		WithEvent("password_reset").
		WithCode("111111").
		Build()

	repo := repository.NewEmailConfirmationRepository(testutil.QueriesWithTx(tx))

	t.Run("IDでメール確認を取得できる", func(t *testing.T) {
		ec, err := repo.FindByID(ctx, id)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}

		if ec.ID != id {
			t.Errorf("ec.ID = %v, want %v", ec.ID, id)
		}
		if ec.Email != email {
			t.Errorf("ec.Email = %v, want %v", ec.Email, email)
		}
	})

	t.Run("存在しないIDはnilを返す", func(t *testing.T) {
		nonExistentID := model.EmailConfirmationID(testutil.MustParseUUID("01234567-89ab-cdef-0123-456789abcdef"))
		ec, err := repo.FindByID(ctx, nonExistentID)
		if err != nil {
			t.Errorf("FindByID() error = %v, want nil", err)
		}
		if ec != nil {
			t.Errorf("FindByID() ec = %v, want nil", ec)
		}
	})
}

func TestEmailConfirmationRepository_FindActiveByID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	repo := repository.NewEmailConfirmationRepository(testutil.QueriesWithTx(tx))

	t.Run("有効期限内かつ未確認のレコードを取得できる", func(t *testing.T) {
		id := testutil.NewEmailConfirmationBuilder(t, tx).
			WithEmail("active-test@example.com").
			WithEvent("password_reset").
			WithCode("222222").
			Build()

		ec, err := repo.FindActiveByID(ctx, id)
		if err != nil {
			t.Fatalf("FindActiveByID() error = %v", err)
		}

		if ec.ID != id {
			t.Errorf("ec.ID = %v, want %v", ec.ID, id)
		}
	})

	t.Run("有効期限切れのレコードはnilを返す", func(t *testing.T) {
		// 20分前に作成されたレコード (15分で期限切れ)
		expiredTime := time.Now().Add(-20 * time.Minute)
		id := testutil.NewEmailConfirmationBuilder(t, tx).
			WithEmail("expired-test@example.com").
			WithEvent("password_reset").
			WithCode("333333").
			WithCreatedAt(expiredTime).
			Build()

		ec, err := repo.FindActiveByID(ctx, id)
		if err != nil {
			t.Errorf("FindActiveByID() error = %v, want nil", err)
		}
		if ec != nil {
			t.Errorf("FindActiveByID() ec = %v, want nil", ec)
		}
	})

	t.Run("確認済みのレコードはnilを返す", func(t *testing.T) {
		succeededAt := time.Now()
		id := testutil.NewEmailConfirmationBuilder(t, tx).
			WithEmail("succeeded-test@example.com").
			WithEvent("password_reset").
			WithCode("444444").
			WithSucceededAt(succeededAt).
			Build()

		ec, err := repo.FindActiveByID(ctx, id)
		if err != nil {
			t.Errorf("FindActiveByID() error = %v, want nil", err)
		}
		if ec != nil {
			t.Errorf("FindActiveByID() ec = %v, want nil", ec)
		}
	})
}

func TestEmailConfirmationRepository_FindSucceededByID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	repo := repository.NewEmailConfirmationRepository(testutil.QueriesWithTx(tx))

	t.Run("確認済みのレコードを取得できる", func(t *testing.T) {
		succeededAt := time.Now()
		id := testutil.NewEmailConfirmationBuilder(t, tx).
			WithEmail("get-succeeded-test@example.com").
			WithEvent("password_reset").
			WithCode("555555").
			WithSucceededAt(succeededAt).
			Build()

		ec, err := repo.FindSucceededByID(ctx, id)
		if err != nil {
			t.Fatalf("FindSucceededByID() error = %v", err)
		}

		if ec.ID != id {
			t.Errorf("ec.ID = %v, want %v", ec.ID, id)
		}
		if ec.SucceededAt == nil {
			t.Error("ec.SucceededAt should not be nil")
		}
	})

	t.Run("未確認のレコードはnilを返す", func(t *testing.T) {
		id := testutil.NewEmailConfirmationBuilder(t, tx).
			WithEmail("not-succeeded-test@example.com").
			WithEvent("password_reset").
			WithCode("666666").
			Build()

		ec, err := repo.FindSucceededByID(ctx, id)
		if err != nil {
			t.Errorf("FindSucceededByID() error = %v, want nil", err)
		}
		if ec != nil {
			t.Errorf("FindSucceededByID() ec = %v, want nil", ec)
		}
	})
}

func TestEmailConfirmationRepository_Succeed(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	repo := repository.NewEmailConfirmationRepository(testutil.QueriesWithTx(tx))

	t.Run("メール確認を成功済みとしてマークできる", func(t *testing.T) {
		id := testutil.NewEmailConfirmationBuilder(t, tx).
			WithEmail("mark-succeeded-test@example.com").
			WithEvent("password_reset").
			WithCode("777777").
			Build()

		// マーク前は未確認
		ec, err := repo.FindByID(ctx, id)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if ec.SucceededAt != nil {
			t.Error("ec.SucceededAt should be nil before marking")
		}

		// 成功済みとしてマーク
		err = repo.Succeed(ctx, id)
		if err != nil {
			t.Fatalf("Succeed() error = %v", err)
		}

		// マーク後は確認済み
		ec, err = repo.FindByID(ctx, id)
		if err != nil {
			t.Fatalf("FindByID() after mark error = %v", err)
		}
		if ec.SucceededAt == nil {
			t.Error("ec.SucceededAt should not be nil after marking")
		}
	})
}
