package usecase_test

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// cleanupOldExportsFixture is the cleanup UseCase wired to a test transaction,
// together with the repository the assertions read rows back with and the
// object storage stand-in.
//
// [Ja] cleanupOldExportsFixture は、テスト用トランザクションに配線した掃除の
// UseCase と、検証で行を読み直す repository、およびオブジェクトストレージの代役。
type cleanupOldExportsFixture struct {
	uc               *usecase.CleanupOldExportsUsecase
	exportRepo       *repository.ExportRepository
	notificationRepo *repository.ExportCompletionNotificationRepository
	storage          *exportDeletionObjectStorage
}

func newCleanupOldExportsFixture(t *testing.T, tx *sql.Tx) *cleanupOldExportsFixture {
	t.Helper()

	queries := testutil.QueriesWithTx(tx)
	exportRepo := repository.NewExportRepository(queries)
	storage := newExportDeletionObjectStorage(t)
	return &cleanupOldExportsFixture{
		uc:               usecase.NewCleanupOldExportsUsecase(exportRepo, storage),
		exportRepo:       exportRepo,
		notificationRepo: repository.NewExportCompletionNotificationRepository(queries),
		storage:          storage,
	}
}

// TestCleanupOldExportsUsecase_Execute pins what the retention policy leaves
// behind. Deleting the latest success would take away the archive the profile
// currently offers for download, and keeping the ones it replaced is the
// storage this cleanup exists to release.
//
// [Ja] TestCleanupOldExportsUsecase_Execute は保持ポリシーが何を残すかを固定する。
// 最新の成功を削除するとプロフィールが現在ダウンロード対象として提供している
// アーカイブを奪うことになり、それが置き換えたものを残すことは、この掃除が解放する
// ために存在するストレージになる。
func TestCleanupOldExportsUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	base := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)

	t.Run("最新の成功を残して置き換えられた成功をオブジェクトごと削除する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newCleanupOldExportsFixture(t, tx)
		target := testutil.NewProfileOwner(t, tx)

		_, old1Key := buildStoredExport(t, tx, fixture.storage, target, model.ExportStatusSucceeded, base)
		_, old2Key := buildStoredExport(t, tx, fixture.storage, target, model.ExportStatusSucceeded, base.Add(time.Hour))
		latest, latestKey := buildStoredExport(t, tx, fixture.storage, target, model.ExportStatusSucceeded, base.Add(2*time.Hour))
		// A failed export and a request that has not been generated yet are not
		// exports the latest success replaced: the failure is what the screen
		// offers a retry from, and the request is the retry already in flight.
		//
		// [Ja] failed のエクスポートと、まだ生成されていない申請は、最新の成功が
		// 置き換えたものではない。失敗は画面が再実行を促すためのものであり、申請は
		// すでに進行中のその再実行である。
		failed, failedKey := buildStoredExport(t, tx, fixture.storage, target, model.ExportStatusFailed, base.Add(3*time.Hour))
		queued, queuedKey := buildStoredExport(t, tx, fixture.storage, target, model.ExportStatusQueued, base.Add(4*time.Hour))

		if err := fixture.uc.Execute(ctx, target.ProfileID); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		// Oldest first, so the deletion follows the order the exports were
		// replaced in.
		//
		// [Ja] 古い順に削除し、エクスポートが置き換えられた順序に従う。
		if got := fixture.storage.deletedKeys(); !slices.Equal(got, []string{old1Key, old2Key}) {
			t.Errorf("削除されたキー = %v, want %v", got, []string{old1Key, old2Key})
		}
		wantStored := []string{latestKey, failedKey, queuedKey}
		slices.Sort(wantStored)
		if got := fixture.storage.storedKeys(); !slices.Equal(got, wantStored) {
			t.Errorf("残っているオブジェクト = %v, want %v", got, wantStored)
		}
		assertExportIDs(t, "残っている行", remainingExportIDs(t, fixture.exportRepo, target.ProfileID), latest, failed, queued)
	})

	t.Run("旧 export 行を削除しても完了通知を outbox から再投入できる", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newCleanupOldExportsFixture(t, tx)
		target := testutil.NewProfileOwner(t, tx)

		old, _ := buildStoredExport(t, tx, fixture.storage, target, model.ExportStatusSucceeded, base)
		buildStoredExport(t, tx, fixture.storage, target, model.ExportStatusSucceeded, base.Add(time.Hour))
		testutil.NewExportCompletionNotificationBuilder(t, tx).
			WithExportID(old).
			WithActorID(target.ActorID).
			WithCreatedAt(base).
			Build()

		if err := fixture.uc.Execute(ctx, target.ProfileID); err != nil {
			t.Fatalf("cleanup Execute() error = %v", err)
		}
		if got, err := fixture.exportRepo.FindByID(ctx, old); err != nil || got != nil {
			t.Fatalf("FindByID() after cleanup = (%v, %v), want (nil, nil)", got, err)
		}
		if got, err := fixture.notificationRepo.FindByExportID(ctx, old); err != nil || got == nil {
			t.Fatalf("FindByExportID() after cleanup = (%v, %v), want (notification, nil)", got, err)
		}

		reconcile := newReconcileExportsFixture(t, tx, usecase.DefaultExportRecoveryLimits())
		if err := reconcile.uc.Execute(ctx); err != nil {
			t.Fatalf("reconcile Execute() error = %v", err)
		}
		assertJobFor(t, reconcile.inserter, "send_export_completed_email", old.String())
	})

	t.Run("オブジェクトが既に無くても行を削除する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newCleanupOldExportsFixture(t, tx)
		target := testutil.NewProfileOwner(t, tx)

		// The object is gone but the row is not: this is where a previous run
		// stopped between the two, and where its rerun has to be able to finish.
		//
		// [Ja] オブジェクトは無いが行は残っている。前回の実行が両者の間で止まった
		// 場合がこれであり、その再実行はここから完了できる必要がある。
		old := testutil.NewExportBuilder(t, tx).
			WithProfileID(target.ProfileID).
			WithActorID(target.ActorID).
			WithStatus(model.ExportStatusSucceeded).
			WithCreatedAt(base).
			Build()
		latest, _ := buildStoredExport(t, tx, fixture.storage, target, model.ExportStatusSucceeded, base.Add(time.Hour))

		if err := fixture.uc.Execute(ctx, target.ProfileID); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		assertExportIDs(t, "残っている行", remainingExportIDs(t, fixture.exportRepo, target.ProfileID), latest)
		if got := fixture.storage.deletedKeys(); !slices.Equal(got, []string{usecase.ExportObjectKey(target.ProfileID, old)}) {
			t.Errorf("削除されたキー = %v", got)
		}
	})

	t.Run("オブジェクトを削除できなければ行を残して失敗を報告し、再実行で完了する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newCleanupOldExportsFixture(t, tx)
		target := testutil.NewProfileOwner(t, tx)

		old1, old1Key := buildStoredExport(t, tx, fixture.storage, target, model.ExportStatusSucceeded, base)
		old2, old2Key := buildStoredExport(t, tx, fixture.storage, target, model.ExportStatusSucceeded, base.Add(time.Hour))
		latest, _ := buildStoredExport(t, tx, fixture.storage, target, model.ExportStatusSucceeded, base.Add(2*time.Hour))
		fixture.storage.failOn(old1Key, errors.New("注入したストレージのエラー"))

		if err := fixture.uc.Execute(ctx, target.ProfileID); err == nil {
			t.Fatal("Execute() = nil, want an error")
		}

		// The row is what names the object, so it stays until the object is gone.
		// Removing it here would leave an object nothing points at, which only the
		// daily orphan sweep could find.
		//
		// [Ja] オブジェクトを指し示すのは行であるため、行はオブジェクトが消えるまで
		// 残る。ここで行を消すと、どこからも指されないオブジェクトが残り、それを
		// 見つけられるのは日次の孤児回収だけになる。
		assertExportIDs(t, "失敗後に残っている行", remainingExportIDs(t, fixture.exportRepo, target.ProfileID), old1, old2, latest)
		if got := fixture.storage.deletedKeys(); len(got) != 0 {
			t.Errorf("削除されたキー = %v, want none", got)
		}

		// The candidate that could not be deleted stays at the head of the list,
		// so the run stops there rather than walking past it: continuing would
		// have every following page start with the same row.
		//
		// [Ja] 削除できなかった候補は一覧の先頭に残るため、実行はそこを飛ばさずに
		// 止まる。続けると以降のどのページも同じ行から始まることになるためである。
		if got := fixture.storage.storedKeys(); !slices.Contains(got, old2Key) {
			t.Errorf("残っているオブジェクト = %v に %s が含まれていない", got, old2Key)
		}

		fixture.storage.failOn(old1Key, nil)
		if err := fixture.uc.Execute(ctx, target.ProfileID); err != nil {
			t.Fatalf("再実行の Execute() error = %v", err)
		}
		assertExportIDs(t, "再実行後に残っている行", remainingExportIDs(t, fixture.exportRepo, target.ProfileID), latest)
	})

	t.Run("成功が最新の 1 件だけなら何も削除しない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newCleanupOldExportsFixture(t, tx)
		target := testutil.NewProfileOwner(t, tx)

		// This is the run enqueued after a profile's first success.
		//
		// [Ja] プロフィールの最初の成功の後に投入される実行がこれにあたる。
		latest, _ := buildStoredExport(t, tx, fixture.storage, target, model.ExportStatusSucceeded, base)

		if err := fixture.uc.Execute(ctx, target.ProfileID); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if got := fixture.storage.deletedKeys(); len(got) != 0 {
			t.Errorf("削除されたキー = %v, want none", got)
		}
		assertExportIDs(t, "残っている行", remainingExportIDs(t, fixture.exportRepo, target.ProfileID), latest)
	})

	t.Run("他プロフィールのエクスポートには触れない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newCleanupOldExportsFixture(t, tx)
		target := testutil.NewProfileOwner(t, tx)
		other := testutil.NewProfileOwner(t, tx)

		_, oldKey := buildStoredExport(t, tx, fixture.storage, target, model.ExportStatusSucceeded, base)
		buildStoredExport(t, tx, fixture.storage, target, model.ExportStatusSucceeded, base.Add(time.Hour))
		otherOld, _ := buildStoredExport(t, tx, fixture.storage, other, model.ExportStatusSucceeded, base)
		otherLatest, _ := buildStoredExport(t, tx, fixture.storage, other, model.ExportStatusSucceeded, base.Add(time.Hour))

		if err := fixture.uc.Execute(ctx, target.ProfileID); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if got := fixture.storage.deletedKeys(); !slices.Equal(got, []string{oldKey}) {
			t.Errorf("削除されたキー = %v, want %v", got, []string{oldKey})
		}
		assertExportIDs(t, "他プロフィールに残っている行", remainingExportIDs(t, fixture.exportRepo, other.ProfileID), otherOld, otherLatest)
	})

	t.Run("実行中に新しい成功が確定しても最新は削除せず置き換えられた分を削除する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newCleanupOldExportsFixture(t, tx)
		target := testutil.NewProfileOwner(t, tx)

		_, oldKey := buildStoredExport(t, tx, fixture.storage, target, model.ExportStatusSucceeded, base)
		_, wasLatestKey := buildStoredExport(t, tx, fixture.storage, target, model.ExportStatusSucceeded, base.Add(time.Hour))

		// A generation finishes while the run is between two candidates, which is
		// what a job argument naming the export to keep would have gone stale on.
		// The candidates are a query, so the export that was the latest becomes a
		// candidate for the same run.
		//
		// [Ja] 実行が候補と候補の間にいる間に生成が完了する。残すエクスポートを名指し
		// するジョブ引数なら、この時点で古くなっている。候補はクエリであるため、最新
		// だったエクスポートは同じ実行の候補になる。
		var newLatest model.ExportID
		var newLatestKey string
		fixture.storage.beforeDelete = func(key string) {
			if key != oldKey {
				return
			}
			newLatest, newLatestKey = buildStoredExport(t, tx, fixture.storage, target, model.ExportStatusSucceeded, base.Add(2*time.Hour))
		}

		if err := fixture.uc.Execute(ctx, target.ProfileID); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if got := fixture.storage.deletedKeys(); !slices.Equal(got, []string{oldKey, wasLatestKey}) {
			t.Errorf("削除されたキー = %v, want %v", got, []string{oldKey, wasLatestKey})
		}
		if got := fixture.storage.storedKeys(); !slices.Equal(got, []string{newLatestKey}) {
			t.Errorf("残っているオブジェクト = %v, want %v", got, []string{newLatestKey})
		}
		assertExportIDs(t, "残っている行", remainingExportIDs(t, fixture.exportRepo, target.ProfileID), newLatest)
	})
}
