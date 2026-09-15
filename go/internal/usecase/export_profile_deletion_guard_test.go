package usecase_test

import (
	"context"

	"github.com/mewstcom/mewst/go/internal/model"
)

// allowingExportProfileDeletionGuardはtransactionに束縛したUseCaseテストの
// 既定guard。productionの調整はcommit済み行を使う並行テストで検証し、これらの
// fixtureは取得済みguardの内側にある処理だけを対象とする。
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
