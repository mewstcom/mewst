package usecase_test

import (
	"context"
	"testing"

	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

func TestDeleteSessionUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("delete-session@example.com").
		Build()

	profileID := testutil.NewProfileBuilder(t, tx).
		WithAtname("deletesessionuser").
		Build()

	actorID := testutil.NewActorBuilder(t, tx).
		WithUserID(userID).
		WithProfileID(profileID).
		Build()

	const token = "delete-session-test-token"

	testutil.NewSessionBuilder(t, tx).
		WithActorID(actorID).
		WithToken(token).
		Build()

	sessionRepo := repository.NewSessionRepository(testutil.QueriesWithTx(tx))
	uc := usecase.NewDeleteSessionUsecase(sessionRepo)

	if err := uc.Execute(ctx, usecase.DeleteSessionInput{Token: token}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	deleted, err := sessionRepo.FindByToken(ctx, token)
	if err != nil {
		t.Fatalf("FindByToken() error = %v", err)
	}
	if deleted != nil {
		t.Error("セッションが削除されていません")
	}
}

func TestDeleteSessionUsecase_Execute_NoMatchingSession(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	sessionRepo := repository.NewSessionRepository(testutil.QueriesWithTx(tx))
	uc := usecase.NewDeleteSessionUsecase(sessionRepo)

	// Deleting a token with no matching row must succeed: sign-out has to stay
	// idempotent for a cookie that points at an already-deleted session.
	//
	// [Ja] 該当行が無いトークンの削除は成功しなければならない。既に削除済みの
	// セッションを指す Cookie でもログアウトが冪等に成立する必要があるため。
	if err := uc.Execute(ctx, usecase.DeleteSessionInput{Token: "nonexistent-token"}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestDeleteSessionUsecase_Execute_EmptyToken(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	sessionRepo := repository.NewSessionRepository(testutil.QueriesWithTx(tx))
	uc := usecase.NewDeleteSessionUsecase(sessionRepo)

	// Roll the transaction back before execution so that any accidental query
	// fails with sql.ErrTxDone. A successful call therefore proves the empty-token
	// guard returned before touching the repository.
	//
	// [Ja] 実行前にトランザクションをロールバックし、誤ってクエリを発行した場合は
	// sql.ErrTxDone で失敗させる。呼び出しが成功すれば、空トークンのガードが
	// Repository に触れる前に return したことを確認できる。
	if err := tx.Rollback(); err != nil {
		t.Fatalf("トランザクションのロールバックに失敗: %v", err)
	}

	// An empty token must succeed without touching the DB, so that callers can
	// hand the request's session token over unconditionally. Sign-out arriving
	// without a session cookie is a normal case, not an error.
	//
	// [Ja] 空のトークンは DB に触れずに成功しなければならない。呼び出し元が
	// リクエストのセッショントークンを無条件に渡せるようにするため。セッション
	// Cookie の無いログアウトは異常系ではなく通常のケースである。
	if err := uc.Execute(ctx, usecase.DeleteSessionInput{Token: ""}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}
