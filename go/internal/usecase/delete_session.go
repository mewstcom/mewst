package usecase

import (
	"context"
	"fmt"

	"github.com/mewstcom/mewst/go/internal/repository"
)

// DeleteSessionUsecase is the use case for deleting a session.
//
// [Ja] DeleteSessionUsecase はセッション削除のユースケース。
type DeleteSessionUsecase struct {
	sessionRepo *repository.SessionRepository
}

// NewDeleteSessionUsecase creates a DeleteSessionUsecase.
//
// [Ja] NewDeleteSessionUsecase は DeleteSessionUsecase を生成する。
func NewDeleteSessionUsecase(sessionRepo *repository.SessionRepository) *DeleteSessionUsecase {
	return &DeleteSessionUsecase{
		sessionRepo: sessionRepo,
	}
}

// DeleteSessionInput holds the input parameters for deleting a session.
//
// [Ja] DeleteSessionInput はセッション削除の入力パラメータ。
type DeleteSessionInput struct {
	Token string
}

// Execute deletes the session that the given token points at.
//
// An empty token returns without issuing a query: a sign-out request that
// carries no session cookie has nothing to delete. A non-empty token with no
// matching row is not an error either, since the DELETE just affects zero rows.
// Callers therefore do not have to check the token or look the session up
// beforehand.
//
// [Ja] Execute は渡されたトークンが指すセッションを削除する。
//
// 空のトークンではクエリを発行せずに戻る。セッション Cookie を持たない
// ログアウトリクエストには削除する対象が無いためである。非空のトークンで
// 該当行が無い場合もエラーにはならず、DELETE の対象が 0 行になるだけである。
// そのため呼び出し元がトークンを検査したりセッションを引いたりする必要はない。
func (uc *DeleteSessionUsecase) Execute(ctx context.Context, input DeleteSessionInput) error {
	if input.Token == "" {
		return nil
	}

	if err := uc.sessionRepo.DeleteByToken(ctx, input.Token); err != nil {
		return fmt.Errorf("セッションの削除に失敗: %w", err)
	}

	return nil
}
