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

// deleteExportsForProfileFixture is the deletion UseCase wired to a test
// transaction, together with the repository the assertions read rows back with
// and the object storage stand-in.
//
// [Ja] deleteExportsForProfileFixture は、テスト用トランザクションに配線した削除の
// UseCase と、検証で行を読み直す repository、およびオブジェクトストレージの代役。
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

// deleteActor deletes the actor row directly and returns what the database
// answered. The exports foreign key covers (actor_id, profile_id), so this is
// where the ON DELETE NO ACTION safety net shows: a caller that forgot to
// remove the exports first is stopped by the database rather than leaving
// archives nothing points at.
//
// The statement runs inside a savepoint because a rejected one aborts the test
// transaction, and the test goes on to assert what happens once the exports are
// gone.
//
// [Ja] deleteActor は actor 行を直接削除し、DB が返したものをそのまま返す。exports
// の外部キーは (actor_id, profile_id) を対象とするため、ON DELETE NO ACTION の
// 安全網が現れるのがここである。先にエクスポートを削除し忘れた呼び出し側は DB に
// 止められ、どこからも指されないアーカイブを残さずに済む。
//
// 文を savepoint の中で実行するのは、拒否された文がテスト用トランザクションを中断
// させるためである。テストはこの後、エクスポートが消えた後の挙動を検証する。
func deleteActor(t *testing.T, tx *sql.Tx, actorID model.ActorID) error {
	t.Helper()

	if _, err := tx.Exec("SAVEPOINT delete_actor"); err != nil {
		t.Fatalf("SAVEPOINT の作成に失敗: %v", err)
	}

	_, err := tx.Exec("DELETE FROM actors WHERE id = $1", uuid.UUID(actorID))
	if err != nil {
		if _, rollbackErr := tx.Exec("ROLLBACK TO SAVEPOINT delete_actor"); rollbackErr != nil {
			t.Fatalf("SAVEPOINT のロールバックに失敗: %v", rollbackErr)
		}
		return err
	}

	if _, err := tx.Exec("RELEASE SAVEPOINT delete_actor"); err != nil {
		t.Fatalf("SAVEPOINT の解放に失敗: %v", err)
	}
	return nil
}

// TestDeleteExportsForProfileUsecase_Execute pins what a profile removal has to
// leave the object storage in. The exports foreign key is NO ACTION precisely
// so that a profile cannot be deleted while its archives are still stored, and
// this use case is what makes that deletion possible without losing track of
// them.
//
// [Ja] TestDeleteExportsForProfileUsecase_Execute は、プロフィールの削除がオブジェクト
// ストレージをどの状態にしなければならないかを固定する。exports の外部キーが NO ACTION
// なのは、アーカイブが保存されたままのプロフィールを削除できないようにするためであり、
// 本 UseCase はそのアーカイブを見失わずに削除を可能にするものである。
func TestDeleteExportsForProfileUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	base := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)

	t.Run("status を問わず全エクスポートをオブジェクトごと削除する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newDeleteExportsForProfileFixture(t, tx)
		target := testutil.NewProfileOwner(t, tx)

		// A failed and a queued export own an object as well: an attempt uploads
		// before the transition that records it, so the key it wrote survives its
		// row's status. A profile removal that only took the succeeded ones would
		// leave those objects stored with nothing naming them.
		//
		// [Ja] failed と queued のエクスポートもオブジェクトを持つ。試行はそれを記録
		// する遷移より先にアップロードするため、書き込まれたキーは行の status より
		// 長く残る。succeeded だけを対象にするプロフィール削除は、それらのオブジェクトを
		// 指すものが無いまま保存し続けることになる。
		_, succeededKey := buildStoredExport(t, tx, fixture.storage, target, model.ExportStatusSucceeded, base)
		_, failedKey := buildStoredExport(t, tx, fixture.storage, target, model.ExportStatusFailed, base.Add(time.Hour))
		_, queuedKey := buildStoredExport(t, tx, fixture.storage, target, model.ExportStatusQueued, base.Add(2*time.Hour))

		if err := fixture.uc.Execute(ctx, target.ProfileID); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		want := []string{succeededKey, failedKey, queuedKey}
		if got := fixture.storage.deletedKeys(); !slices.Equal(got, want) {
			t.Errorf("削除されたキー = %v, want %v", got, want)
		}
		if got := fixture.storage.storedKeys(); len(got) != 0 {
			t.Errorf("残っているオブジェクト = %v, want none", got)
		}
		assertExportIDs(t, "残っている行", remainingExportIDs(t, fixture.exportRepo, target.ProfileID))
	})

	// The notification outlives its export on purpose, so retention cleanup can
	// delete an old export without losing the email it still owes. A profile
	// being deleted owes none, and the row would otherwise announce an archive
	// that no longer exists until the actor deletion cascades it away.
	//
	// [Ja] 通知が export より長く残るのは意図した設計で、保持 cleanup が旧
	// エクスポートを削除しても未送信のメールを失わないようにするためである。削除される
	// プロフィールに送るべきメールは無く、この行を残すと、actor の削除が CASCADE で
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

		// The notification of a profile that is not being deleted stays, even
		// though its export row is untouched by this run either.
		//
		// [Ja] 削除対象ではないプロフィールの通知は残る。その export 行も本実行が
		// 触れないのと同様である。
		otherExport, _ := buildStoredExport(t, tx, fixture.storage, other, model.ExportStatusSucceeded, base)
		testutil.NewExportCompletionNotificationBuilder(t, tx).
			WithExportID(otherExport).
			WithActorID(other.ActorID).
			WithCreatedAt(base).
			Build()

		if err := fixture.uc.Execute(ctx, target.ProfileID); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if got, err := fixture.notificationRepo.FindByExportID(ctx, export); err != nil || got != nil {
			t.Errorf("FindByExportID() after delete = (%v, %v), want (nil, nil)", got, err)
		}
		if got, err := fixture.notificationRepo.FindByExportID(ctx, otherExport); err != nil || got == nil {
			t.Errorf("他プロフィールの FindByExportID() = (%v, %v), want (notification, nil)", got, err)
		}
	})

	// A failure has to stop before the cancellation: the caller aborts the
	// profile deletion, so the exports it could not remove keep the email they
	// still owe.
	//
	// [Ja] 失敗は取り消しの前で止まる必要がある。呼び出し側はプロフィールの削除を
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
			t.Fatal("Execute() = nil, want an error")
		}
		if got, err := fixture.notificationRepo.FindByExportID(ctx, export); err != nil || got == nil {
			t.Errorf("FindByExportID() after failure = (%v, %v), want (notification, nil)", got, err)
		}
	})

	t.Run("エクスポートが残る間は actor を削除できず、削除後は削除できる", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newDeleteExportsForProfileFixture(t, tx)
		target := testutil.NewProfileOwner(t, tx)

		buildStoredExport(t, tx, fixture.storage, target, model.ExportStatusSucceeded, base)

		if err := deleteActor(t, tx, target.ActorID); err == nil {
			t.Fatal("エクスポートが残る状態で actor を削除できてしまった (NO ACTION が効いていない)")
		}

		if err := fixture.uc.Execute(ctx, target.ProfileID); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		// The safety net only holds if the deletion actually satisfies it. A run
		// that removed the objects but left a row would keep the parent
		// undeletable, which is the failure this assertion catches.
		//
		// [Ja] 安全網が機能するのは、削除が実際にそれを満たす場合だけである。
		// オブジェクトを消して行を残した実行は親を削除できないままにする。この検証が
		// 捉えるのはその失敗である。
		if err := deleteActor(t, tx, target.ActorID); err != nil {
			t.Errorf("エクスポート削除後の actor の削除に失敗: %v", err)
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

		// The caller must see the failure: removing the profile after this run
		// would be removing it while its archives are still stored.
		//
		// [Ja] 呼び出し側は失敗を知る必要がある。この実行の後でプロフィールを削除する
		// ことは、アーカイブが保存されたまま削除することになるため。
		if err := fixture.uc.Execute(ctx, target.ProfileID); err == nil {
			t.Fatal("Execute() = nil, want an error")
		}
		assertExportIDs(t, "失敗後に残っている行", remainingExportIDs(t, fixture.exportRepo, target.ProfileID), oldest, newest)

		fixture.storage.failOn(oldestKey, nil)
		if err := fixture.uc.Execute(ctx, target.ProfileID); err != nil {
			t.Fatalf("再実行の Execute() error = %v", err)
		}
		assertExportIDs(t, "再実行後に残っている行", remainingExportIDs(t, fixture.exportRepo, target.ProfileID))
		if got := fixture.storage.storedKeys(); len(got) != 0 {
			t.Errorf("残っているオブジェクト = %v, want none", got)
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
			t.Fatalf("Execute() error = %v", err)
		}

		if got := fixture.storage.deletedKeys(); !slices.Equal(got, []string{targetKey}) {
			t.Errorf("削除されたキー = %v, want %v", got, []string{targetKey})
		}
		if got := fixture.storage.storedKeys(); !slices.Equal(got, []string{otherKey}) {
			t.Errorf("残っているオブジェクト = %v, want %v", got, []string{otherKey})
		}
		assertExportIDs(t, "他プロフィールに残っている行", remainingExportIDs(t, fixture.exportRepo, other.ProfileID), otherExport)
	})

	t.Run("エクスポートを持たないプロフィールでもエラーにならない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		fixture := newDeleteExportsForProfileFixture(t, tx)
		target := testutil.NewProfileOwner(t, tx)

		if err := fixture.uc.Execute(ctx, target.ProfileID); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if got := fixture.storage.deletedKeys(); len(got) != 0 {
			t.Errorf("削除されたキー = %v, want none", got)
		}
	})
}
