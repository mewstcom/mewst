package usecase

import (
	"context"
	"fmt"

	"github.com/mewstcom/mewst/go/internal/repository"
)

// DeleteSessionUsecaseはセッション削除のユースケース。
type DeleteSessionUsecase struct {
	sessionRepo *repository.SessionRepository
}

// NewDeleteSessionUsecaseはDeleteSessionUsecaseを生成する。
func NewDeleteSessionUsecase(sessionRepo *repository.SessionRepository) *DeleteSessionUsecase {
	return &DeleteSessionUsecase{
		sessionRepo: sessionRepo,
	}
}

// DeleteSessionInputはセッション削除の入力パラメータ。
type DeleteSessionInput struct {
	Token string
}

// Executeは渡されたトークンが指すセッションを削除する。
//
// 空のトークンではクエリを発行せずに戻る。セッションCookieを持たない
// ログアウトリクエストには削除する対象が無いためである。非空のトークンで
// 該当行が無い場合もエラーにはならず、DELETEの対象が0行になるだけである。
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
