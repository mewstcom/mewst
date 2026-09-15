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
		t.Fatalf("Execute()のエラー = %v", err)
	}

	deleted, err := sessionRepo.FindByToken(ctx, token)
	if err != nil {
		t.Fatalf("FindByToken()のエラー = %v", err)
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

	// 該当行が無いトークンの削除は成功しなければならない。既に削除済みの
	// セッションを指すCookieでもログアウトが冪等に成立する必要があるため。
	if err := uc.Execute(ctx, usecase.DeleteSessionInput{Token: "nonexistent-token"}); err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}
}

func TestDeleteSessionUsecase_Execute_EmptyToken(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	sessionRepo := repository.NewSessionRepository(testutil.QueriesWithTx(tx))
	uc := usecase.NewDeleteSessionUsecase(sessionRepo)

	// 実行前にトランザクションをロールバックし、誤ってクエリを発行した場合は
	// sql.ErrTxDoneで失敗させる。呼び出しが成功すれば、空トークンのガードが
	// Repositoryに触れる前にreturnしたことを確認できる。
	if err := tx.Rollback(); err != nil {
		t.Fatalf("トランザクションのロールバックに失敗: %v", err)
	}

	// 空のトークンはDBに触れずに成功しなければならない。呼び出し元が
	// リクエストのセッショントークンを無条件に渡せるようにするため。セッション
	// Cookieの無いログアウトは異常系ではなく通常のケースである。
	if err := uc.Execute(ctx, usecase.DeleteSessionInput{Token: ""}); err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}
}
