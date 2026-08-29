package usecase_test

import (
	"context"

	"github.com/mewstcom/mewst/go/internal/model"
)

// allowingExportProfileDeletionGuard is the default guard for transaction-bound
// UseCase tests. The production coordination is covered by committed-row
// concurrency tests; these fixtures only need to exercise the work inside an
// already acquired guard.
//
// [Ja] allowingExportProfileDeletionGuard は transaction に束縛した UseCase テストの
// 既定 guard。production の調整は commit 済み行を使う並行テストで検証し、これらの
// fixture は取得済み guard の内側にある処理だけを対象とする。
type allowingExportProfileDeletionGuard struct{}

func (allowingExportProfileDeletionGuard) BeginOperation(
	context.Context,
	model.ProfileID,
) (func() error, bool, error) {
	return func() error { return nil }, true, nil
}

func (allowingExportProfileDeletionGuard) BeginDeletion(
	context.Context,
	model.ProfileID,
) (func() error, bool, error) {
	return func() error { return nil }, true, nil
}
