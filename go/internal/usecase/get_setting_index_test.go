package usecase_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

func newGetSettingIndexUsecase(t *testing.T, tx *sql.Tx) *usecase.GetSettingIndexUsecase {
	t.Helper()

	return usecase.NewGetSettingIndexUsecase(repository.NewFeatureFlagRepository(testutil.QueriesWithTx(tx)))
}

// TestGetSettingIndexUsecase_Execute pins which grants put the export entry in
// an actor's settings menu.
//
// [Ja] TestGetSettingIndexUsecase_Execute は、どの付与が actor の設定メニューに
// エクスポート項目を載せるのかを固定する。
func TestGetSettingIndexUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("フラグを持たない actor では false を返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		owner := testutil.NewProfileOwner(t, tx)
		uc := newGetSettingIndexUsecase(t, tx)

		output, err := uc.Execute(ctx, usecase.GetSettingIndexInput{ActorID: owner.ActorID})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if output.ExportEnabled {
			t.Error("ExportEnabled = true, want false")
		}
	})

	t.Run("go_export を付与された actor では true を返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		owner := testutil.NewProfileOwner(t, tx)
		testutil.NewFeatureFlagBuilder(t, tx).
			WithActorID(owner.ActorID).
			WithName(model.FeatureFlagExport).
			Build()
		uc := newGetSettingIndexUsecase(t, tx)

		output, err := uc.Execute(ctx, usecase.GetSettingIndexInput{ActorID: owner.ActorID})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if !output.ExportEnabled {
			t.Error("ExportEnabled = false, want true")
		}
	})

	t.Run("別の actor への付与では false を返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		owner := testutil.NewProfileOwner(t, tx)
		other := testutil.NewProfileOwner(t, tx)
		testutil.NewFeatureFlagBuilder(t, tx).
			WithActorID(other.ActorID).
			WithName(model.FeatureFlagExport).
			Build()
		uc := newGetSettingIndexUsecase(t, tx)

		output, err := uc.Execute(ctx, usecase.GetSettingIndexInput{ActorID: owner.ActorID})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if output.ExportEnabled {
			t.Error("ExportEnabled = true, want false")
		}
	})

	t.Run("別のフラグ名の付与では false を返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		owner := testutil.NewProfileOwner(t, tx)
		testutil.NewFeatureFlagBuilder(t, tx).
			WithActorID(owner.ActorID).
			WithName(model.FeatureFlagExample).
			Build()
		uc := newGetSettingIndexUsecase(t, tx)

		output, err := uc.Execute(ctx, usecase.GetSettingIndexInput{ActorID: owner.ActorID})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if output.ExportEnabled {
			t.Error("ExportEnabled = true, want false")
		}
	})

	// A device_token grant routes /settings/export to the Go version for that
	// browser, but the menu decides per actor. Pinning it here keeps the
	// difference deliberate rather than an unnoticed change of behaviour.
	//
	// [Ja] device_token による付与は、そのブラウザの /settings/export を Go 版へ
	// 振り分けるが、メニューは actor 単位で判定する。ここで固定することで、この差を
	// 気づかれない挙動変化ではなく意図した設計として保つ。
	t.Run("device_token だけへの付与では false を返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		owner := testutil.NewProfileOwner(t, tx)
		testutil.NewFeatureFlagBuilder(t, tx).
			WithDeviceToken("test-device-token").
			WithName(model.FeatureFlagExport).
			Build()
		uc := newGetSettingIndexUsecase(t, tx)

		output, err := uc.Execute(ctx, usecase.GetSettingIndexInput{ActorID: owner.ActorID})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if output.ExportEnabled {
			t.Error("ExportEnabled = true, want false")
		}
	})
}
