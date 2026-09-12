package usecase

import (
	"context"

	"github.com/mewstcom/mewst/go/internal/model"
)

// ExportProfileDeletionGuard coordinates export operations and deletion for
// one profile. An allowed/found operation owns the returned release function
// until all of its database, object-storage, and external-side-effect work is
// complete.
//
// [Ja] ExportProfileDeletionGuard は 1 プロフィールの export 操作と削除を調整する。
// allowed / found となった操作は、DB・オブジェクトストレージ・外部副作用の処理が
// すべて終わるまで、返された release 関数を保持する。
type ExportProfileDeletionGuard interface {
	BeginOperation(ctx context.Context, profileID model.ProfileID) (release func() error, allowed bool, err error)
	BeginDeletion(ctx context.Context, profileID model.ProfileID) (release func() error, found bool, err error)
}
