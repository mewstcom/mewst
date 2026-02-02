package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/mewstcom/mewst/internal/model"
	"github.com/mewstcom/mewst/internal/repository"
	"github.com/mewstcom/mewst/internal/testutil"
)

func TestEmailConfirmationRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	repo := repository.NewEmailConfirmationRepository(tx)

	t.Run("メール確認を作成できる", func(t *testing.T) {
		params := repository.CreateEmailConfirmationParams{
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

func TestEmailConfirmationRepository_GetByID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	email := "getbyid-test@example.com"
	id := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail(email).
		WithEvent("password_reset").
		WithCode("111111").
		Build()

	repo := repository.NewEmailConfirmationRepository(tx)

	t.Run("IDでメール確認を取得できる", func(t *testing.T) {
		ec, err := repo.GetByID(ctx, id)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}

		if ec.ID != id {
			t.Errorf("ec.ID = %v, want %v", ec.ID, id)
		}
		if ec.Email != email {
			t.Errorf("ec.Email = %v, want %v", ec.Email, email)
		}
	})

	t.Run("存在しないIDはErrNotFoundを返す", func(t *testing.T) {
		nonExistentID := testutil.MustParseUUID("01234567-89ab-cdef-0123-456789abcdef")
		_, err := repo.GetByID(ctx, nonExistentID)
		if err == nil {
			t.Error("GetByID() should return error for non-existent ID")
		}
		if err != repository.ErrNotFound {
			t.Errorf("GetByID() error = %v, want ErrNotFound", err)
		}
	})
}

func TestEmailConfirmationRepository_GetActiveByID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	repo := repository.NewEmailConfirmationRepository(tx)

	t.Run("有効期限内かつ未確認のレコードを取得できる", func(t *testing.T) {
		id := testutil.NewEmailConfirmationBuilder(t, tx).
			WithEmail("active-test@example.com").
			WithEvent("password_reset").
			WithCode("222222").
			Build()

		ec, err := repo.GetActiveByID(ctx, id)
		if err != nil {
			t.Fatalf("GetActiveByID() error = %v", err)
		}

		if ec.ID != id {
			t.Errorf("ec.ID = %v, want %v", ec.ID, id)
		}
	})

	t.Run("有効期限切れのレコードはErrNotFoundを返す", func(t *testing.T) {
		// 20分前に作成されたレコード（15分で期限切れ）
		expiredTime := time.Now().Add(-20 * time.Minute)
		id := testutil.NewEmailConfirmationBuilder(t, tx).
			WithEmail("expired-test@example.com").
			WithEvent("password_reset").
			WithCode("333333").
			WithCreatedAt(expiredTime).
			Build()

		_, err := repo.GetActiveByID(ctx, id)
		if err == nil {
			t.Error("GetActiveByID() should return error for expired record")
		}
		if err != repository.ErrNotFound {
			t.Errorf("GetActiveByID() error = %v, want ErrNotFound", err)
		}
	})

	t.Run("確認済みのレコードはErrNotFoundを返す", func(t *testing.T) {
		succeededAt := time.Now()
		id := testutil.NewEmailConfirmationBuilder(t, tx).
			WithEmail("succeeded-test@example.com").
			WithEvent("password_reset").
			WithCode("444444").
			WithSucceededAt(succeededAt).
			Build()

		_, err := repo.GetActiveByID(ctx, id)
		if err == nil {
			t.Error("GetActiveByID() should return error for succeeded record")
		}
		if err != repository.ErrNotFound {
			t.Errorf("GetActiveByID() error = %v, want ErrNotFound", err)
		}
	})
}

func TestEmailConfirmationRepository_GetSucceededByID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	repo := repository.NewEmailConfirmationRepository(tx)

	t.Run("確認済みのレコードを取得できる", func(t *testing.T) {
		succeededAt := time.Now()
		id := testutil.NewEmailConfirmationBuilder(t, tx).
			WithEmail("get-succeeded-test@example.com").
			WithEvent("password_reset").
			WithCode("555555").
			WithSucceededAt(succeededAt).
			Build()

		ec, err := repo.GetSucceededByID(ctx, id)
		if err != nil {
			t.Fatalf("GetSucceededByID() error = %v", err)
		}

		if ec.ID != id {
			t.Errorf("ec.ID = %v, want %v", ec.ID, id)
		}
		if ec.SucceededAt == nil {
			t.Error("ec.SucceededAt should not be nil")
		}
	})

	t.Run("未確認のレコードはErrNotFoundを返す", func(t *testing.T) {
		id := testutil.NewEmailConfirmationBuilder(t, tx).
			WithEmail("not-succeeded-test@example.com").
			WithEvent("password_reset").
			WithCode("666666").
			Build()

		_, err := repo.GetSucceededByID(ctx, id)
		if err == nil {
			t.Error("GetSucceededByID() should return error for not succeeded record")
		}
		if err != repository.ErrNotFound {
			t.Errorf("GetSucceededByID() error = %v, want ErrNotFound", err)
		}
	})
}

func TestEmailConfirmationRepository_MarkAsSucceeded(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	ctx := context.Background()

	repo := repository.NewEmailConfirmationRepository(tx)

	t.Run("メール確認を成功済みとしてマークできる", func(t *testing.T) {
		id := testutil.NewEmailConfirmationBuilder(t, tx).
			WithEmail("mark-succeeded-test@example.com").
			WithEvent("password_reset").
			WithCode("777777").
			Build()

		// マーク前は未確認
		ec, err := repo.GetByID(ctx, id)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}
		if ec.SucceededAt != nil {
			t.Error("ec.SucceededAt should be nil before marking")
		}

		// 成功済みとしてマーク
		err = repo.MarkAsSucceeded(ctx, id)
		if err != nil {
			t.Fatalf("MarkAsSucceeded() error = %v", err)
		}

		// マーク後は確認済み
		ec, err = repo.GetByID(ctx, id)
		if err != nil {
			t.Fatalf("GetByID() after mark error = %v", err)
		}
		if ec.SucceededAt == nil {
			t.Error("ec.SucceededAt should not be nil after marking")
		}
	})
}
