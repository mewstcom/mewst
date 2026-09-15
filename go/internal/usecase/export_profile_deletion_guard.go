package usecase

import (
	"context"

	"github.com/mewstcom/mewst/go/internal/model"
)

// ExportProfileDeletionGuardは1プロフィールのexport操作と削除を調整する。
// allowed / foundとなった操作は、DB・オブジェクトストレージ・外部副作用の処理が
// すべて終わるまで、返されたrelease関数を保持する。
type ExportProfileDeletionGuard interface {
	BeginOperation(ctx context.Context, profileID model.ProfileID) (release func() error, allowed bool, err error)
	BeginDeletion(ctx context.Context, profileID model.ProfileID) (release func() error, found bool, err error)
}
