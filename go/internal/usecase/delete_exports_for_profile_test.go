package usecase_test

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// deleteExportsForProfileFixtureは、テスト用トランザクションに配線した削除の
// UseCaseと、検証で行を読み直すrepository、およびオブジェクトストレージの代役。
type deleteExportsForProfileFixture struct {
	uc               *usecase.DeleteExportsForProfileUsecase
	exportRepo       *repository.ExportRepository
	notificationRepo *repository.ExportCompletionNotificationRepository
	storage          *exportDeletionObjectStorage
}

func newDeleteExportsForProfileFixture(t *testing.T, tx *sql.Tx) *deleteExportsForProfileFixture {
	t.Helper()

	queries := testutil.QueriesWithTx(tx)
	exportRepo := repository.NewExportRepository(queries)
	notificationRepo := repository.NewExportCompletionNotificationRepository(queries)
	storage := newExportDeletionObjectStorage(t)
	return &deleteExportsForProfileFixture{
		uc: usecase.NewDeleteExportsForProfileUsecase(
			exportRepo,
			notificationRepo,
			allowingExportProfileDeletionGuard{},
			storage,
		),
		exportRepo:       exportRepo,
		notificationRepo: notificationRepo,
		storage:          storage,
	}
}

// deleteActorはactor行を直接削除し、DBが返したものをそのまま返す。exports
// の外部キーは (actor_id, profile_id) を対象とするため、ON DELETE NO ACTIONの
// 安全網が現れるのがここである。先にエクスポートを削除し忘れた呼び出し側はDBに
// 止められ、どこからも指されないアーカイブを残さずに済む。
//
// 文をsavepointの中で実行するのは、拒否された文がテスト用トランザクションを中断
// させるためである。テストはこの後、エクスポートが消えた後の挙動を検証する。
func deleteActor(t *testing.T, tx *sql.Tx, actorID model.ActorID) error {
	t.Helper()

	if _, err := tx.Exec("SAVEPOINT delete_actor"); err != nil {
		t.Fatalf("SAVEPOINTの作成に失敗: %v", err)
	}

	_, err := tx.Exec("DELETE FROM actors WHERE id = $1", uuid.UUID(actorID))
	if err != nil {
		if _, rollbackErr := tx.Exec("ROLLBACK TO SAVEPOINT delete_actor"); rollbackErr != nil {
			t.Fatalf("SAVEPOINTのロールバックに失敗: %v", rollbackErr)
		}
		return err
	}

	if _, err := tx.Exec("RELEASE SAVEPOINT delete_actor"); err != nil {
		t.Fatalf("SAVEPOINTの解放に失敗: %v", err)
	}
	return nil
}

// TestDeleteExportsForProfileUsecase_Executeは、プロフィールの削除がオブジェクト
// ストレージをどの状態にしなければならないかを固定する。exportsの外部キーがNO ACTION
// なのは、アーカイブが保存されたままのプロフィールを削除できないようにするためであり、
// 本UseCaseはそのアーカイブを見失わずに削除を可能にするものである。
func TestDeleteExportsForProfileUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	base := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)

	t.Run("statusを問わず全エクスポートをオブジェクトごと削除する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newDeleteExportsForProfileFixture(t, tx)
		target := testutil.NewProfileOwner(t, tx)

		// failedとqueuedのエクスポートもオブジェクトを持つ。試行はそれを記録
		// する遷移より先にアップロードするため、書き込まれたキーは行のstatusより
		// 長く残る。succeededだけを対象にするプロフィール削除は、それらのオブジェクトを
		// 指すものが無いまま保存し続けることになる。
		_, succeededKey := buildStoredExport(t, tx, fixture.storage, target, model.ExportStatusSucceeded, base)
		_, failedKey := buildStoredExport(t, tx, fixture.storage, target, model.ExportStatusFailed, base.Add(time.Hour))
		_, queuedKey := buildStoredExport(t, tx, fixture.storage, target, model.ExportStatusQueued, base.Add(2*time.Hour))

		if err := fixture.uc.Execute(ctx, target.ProfileID); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		want := []string{succeededKey, failedKey, queuedKey}
		if got := fixture.storage.deletedKeys(); !slices.Equal(got, want) {
			t.Errorf("削除されたキー = %v、期待値 = %v", got, want)
		}
		if got := fixture.storage.storedKeys(); len(got) != 0 {
			t.Errorf("残っているオブジェクト = %v、期待値 = なし", got)
		}
		assertExportIDs(t, "残っている行", remainingExportIDs(t, fixture.exportRepo, target.ProfileID))
	})

	// 通知がexportより長く残るのは意図した設計で、保持cleanupが旧
	// エクスポートを削除しても未送信のメールを失わないようにするためである。削除される
	// プロフィールに送るべきメールは無く、この行を残すと、actorの削除がCASCADEで
	// 消すまでの間、すでに存在しないアーカイブを知らせることになる。
	t.Run("送信待ちの完了通知も取り消す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newDeleteExportsForProfileFixture(t, tx)
		target := testutil.NewProfileOwner(t, tx)
		other := testutil.NewProfileOwner(t, tx)

		export, _ := buildStoredExport(t, tx, fixture.storage, target, model.ExportStatusSucceeded, base)
		testutil.NewExportCompletionNotificationBuilder(t, tx).
			WithExportID(export).
			WithActorID(target.ActorID).
			WithCreatedAt(base).
			Build()

		// 削除対象ではないプロフィールの通知は残る。そのexport行も本実行が
		// 触れないのと同様である。
		otherExport, _ := buildStoredExport(t, tx, fixture.storage, other, model.ExportStatusSucceeded, base)
		testutil.NewExportCompletionNotificationBuilder(t, tx).
			WithExportID(otherExport).
			WithActorID(other.ActorID).
			WithCreatedAt(base).
			Build()

		if err := fixture.uc.Execute(ctx, target.ProfileID); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		if got, err := fixture.notificationRepo.FindByExportID(ctx, export); err != nil || got != nil {
			t.Errorf("削除後のFindByExportID() = (%v, %v)、期待値 = (nil, nil)", got, err)
		}
		if got, err := fixture.notificationRepo.FindByExportID(ctx, otherExport); err != nil || got == nil {
			t.Errorf("他プロフィールのFindByExportID() = (%v, %v)、期待値 = (通知, nil)", got, err)
		}
	})

	// 失敗は取り消しの前で止まる必要がある。呼び出し側はプロフィールの削除を
	// 中断するため、削除できなかったエクスポートは未送信のメールを保持し続ける。
	t.Run("エクスポートの削除に失敗したら完了通知を取り消さない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newDeleteExportsForProfileFixture(t, tx)
		target := testutil.NewProfileOwner(t, tx)

		export, exportKey := buildStoredExport(t, tx, fixture.storage, target, model.ExportStatusSucceeded, base)
		testutil.NewExportCompletionNotificationBuilder(t, tx).
			WithExportID(export).
			WithActorID(target.ActorID).
			WithCreatedAt(base).
			Build()
		fixture.storage.failOn(exportKey, errors.New("注入したストレージのエラー"))

		if err := fixture.uc.Execute(ctx, target.ProfileID); err == nil {
			t.Fatal("Execute()のエラー = nil、エラーを期待")
		}
		if got, err := fixture.notificationRepo.FindByExportID(ctx, export); err != nil || got == nil {
			t.Errorf("失敗後のFindByExportID() = (%v, %v)、期待値 = (通知, nil)", got, err)
		}
	})

	t.Run("エクスポートが残る間はactorを削除できず、削除後は削除できる", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newDeleteExportsForProfileFixture(t, tx)
		target := testutil.NewProfileOwner(t, tx)

		buildStoredExport(t, tx, fixture.storage, target, model.ExportStatusSucceeded, base)

		if err := deleteActor(t, tx, target.ActorID); err == nil {
			t.Fatal("エクスポートが残る状態でactorを削除できてしまった (NO ACTIONが効いていない)")
		}

		if err := fixture.uc.Execute(ctx, target.ProfileID); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		// 安全網が機能するのは、削除が実際にそれを満たす場合だけである。
		// オブジェクトを消して行を残した実行は親を削除できないままにする。この検証が
		// 捉えるのはその失敗である。
		if err := deleteActor(t, tx, target.ActorID); err != nil {
			t.Errorf("エクスポート削除後のactorの削除に失敗: %v", err)
		}
	})

	t.Run("オブジェクトを削除できなければ行を残して失敗を報告し、再実行で完了する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newDeleteExportsForProfileFixture(t, tx)
		target := testutil.NewProfileOwner(t, tx)

		oldest, oldestKey := buildStoredExport(t, tx, fixture.storage, target, model.ExportStatusSucceeded, base)
		newest, _ := buildStoredExport(t, tx, fixture.storage, target, model.ExportStatusSucceeded, base.Add(time.Hour))
		fixture.storage.failOn(oldestKey, errors.New("注入したストレージのエラー"))

		// 呼び出し側は失敗を知る必要がある。この実行の後でプロフィールを削除する
		// ことは、アーカイブが保存されたまま削除することになるため。
		if err := fixture.uc.Execute(ctx, target.ProfileID); err == nil {
			t.Fatal("Execute()のエラー = nil、エラーを期待")
		}
		assertExportIDs(t, "失敗後に残っている行", remainingExportIDs(t, fixture.exportRepo, target.ProfileID), oldest, newest)

		fixture.storage.failOn(oldestKey, nil)
		if err := fixture.uc.Execute(ctx, target.ProfileID); err != nil {
			t.Fatalf("再実行のExecute()のエラー = %v", err)
		}
		assertExportIDs(t, "再実行後に残っている行", remainingExportIDs(t, fixture.exportRepo, target.ProfileID))
		if got := fixture.storage.storedKeys(); len(got) != 0 {
			t.Errorf("残っているオブジェクト = %v、期待値 = なし", got)
		}
	})

	t.Run("他プロフィールのエクスポートには触れない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newDeleteExportsForProfileFixture(t, tx)
		target := testutil.NewProfileOwner(t, tx)
		other := testutil.NewProfileOwner(t, tx)

		_, targetKey := buildStoredExport(t, tx, fixture.storage, target, model.ExportStatusSucceeded, base)
		otherExport, otherKey := buildStoredExport(t, tx, fixture.storage, other, model.ExportStatusSucceeded, base)

		if err := fixture.uc.Execute(ctx, target.ProfileID); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		if got := fixture.storage.deletedKeys(); !slices.Equal(got, []string{targetKey}) {
			t.Errorf("削除されたキー = %v、期待値 = %v", got, []string{targetKey})
		}
		if got := fixture.storage.storedKeys(); !slices.Equal(got, []string{otherKey}) {
			t.Errorf("残っているオブジェクト = %v、期待値 = %v", got, []string{otherKey})
		}
		assertExportIDs(t, "他プロフィールに残っている行", remainingExportIDs(t, fixture.exportRepo, other.ProfileID), otherExport)
	})

	t.Run("エクスポートを持たないプロフィールでもエラーにならない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newDeleteExportsForProfileFixture(t, tx)
		target := testutil.NewProfileOwner(t, tx)

		if err := fixture.uc.Execute(ctx, target.ProfileID); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if got := fixture.storage.deletedKeys(); len(got) != 0 {
			t.Errorf("削除されたキー = %v、期待値 = なし", got)
		}
	})
}
