package usecase_test

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// countingJobInserter records the jobs it was asked to insert and can be made
// to fail. Unlike recordingJobInserter it guards its state with a mutex,
// because the concurrent-create test drives two Execute calls at once.
//
// [Ja] countingJobInserter は投入を依頼されたジョブを記録し、失敗させることも
// できる。recordingJobInserter と違い状態を mutex で守るのは、同時 Create の
// テストが 2 つの Execute を同時に走らせるためである。
type countingJobInserter struct {
	mu      sync.Mutex
	inserts []river.JobArgs
	err     error
}

func (i *countingJobInserter) Insert(_ context.Context, args river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.err != nil {
		return nil, i.err
	}
	i.inserts = append(i.inserts, args)
	return &rivertype.JobInsertResult{}, nil
}

func (i *countingJobInserter) generateExportIDs() []string {
	i.mu.Lock()
	defer i.mu.Unlock()

	var ids []string
	for _, args := range i.inserts {
		if generate, ok := args.(dispatcher.GenerateExportArgs); ok {
			ids = append(ids, generate.ExportID)
		}
	}
	return ids
}

// newCreateExportUsecase builds a CreateExportUsecase whose repositories run
// against the shared test DB. It opens its own transaction, so prerequisite
// rows have to be committed rather than held in an outer transaction.
//
// [Ja] newCreateExportUsecase は共有テスト DB に対して動く CreateExportUsecase を
// 構築する。この UseCase は自身の transaction を開くため、前提となる行はアウターの
// transaction に保持したままではなく commit しておく必要がある。
func newCreateExportUsecase(db *sql.DB, inserter dispatcher.JobInserter, storageReady bool) *usecase.CreateExportUsecase {
	q := query.New(db)
	return usecase.NewCreateExportUsecase(
		db,
		repository.NewUserProfileRepository(q),
		repository.NewExportRepository(q),
		dispatcher.NewDispatcher(inserter),
		storageReady,
	)
}

// committedExportOwner is a user owning a profile, committed to the shared test
// DB so that the UseCase's own transaction can see it.
//
// [Ja] committedExportOwner はプロフィールを所有するユーザーで、UseCase 自身の
// transaction から見えるよう共有テスト DB へ commit してある。
type committedExportOwner struct {
	testutil.ProfileOwner
	db *sql.DB
}

func newCommittedExportOwner(t *testing.T) committedExportOwner {
	t.Helper()

	db := testutil.GetTestDB()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("前提データ用 transaction の開始に失敗: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	owner := testutil.NewProfileOwner(t, tx)
	if err := tx.Commit(); err != nil {
		t.Fatalf("前提データの commit に失敗: %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM export_completion_notifications WHERE profile_id = $1", uuid.UUID(owner.ProfileID))
		_, _ = db.Exec("DELETE FROM export_posts WHERE export_id IN (SELECT id FROM exports WHERE profile_id = $1)", uuid.UUID(owner.ProfileID))
		_, _ = db.Exec("DELETE FROM exports WHERE profile_id = $1", uuid.UUID(owner.ProfileID))
		_, _ = db.Exec("DELETE FROM actors WHERE id = $1", uuid.UUID(owner.ActorID))
		_, _ = db.Exec("DELETE FROM user_profiles WHERE profile_id = $1", uuid.UUID(owner.ProfileID))
		_, _ = db.Exec("DELETE FROM profiles WHERE id = $1", uuid.UUID(owner.ProfileID))
		_, _ = db.Exec("DELETE FROM users WHERE id = $1", uuid.UUID(owner.UserID))
	})

	return committedExportOwner{ProfileOwner: owner, db: db}
}

// commitExport creates one export with the given status for the owner and
// commits it, so the UseCase's transaction observes it as existing state.
//
// [Ja] commitExport は指定 status のエクスポートを 1 件その所有者に作って commit
// する。UseCase の transaction が既存の状態として観測できるようにするため。
func commitExport(t *testing.T, owner committedExportOwner, status model.ExportStatus) model.ExportID {
	t.Helper()

	tx, err := owner.db.Begin()
	if err != nil {
		t.Fatalf("エクスポート作成用 transaction の開始に失敗: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	id := testutil.NewExportBuilder(t, tx).
		WithProfileID(owner.ProfileID).
		WithActorID(owner.ActorID).
		WithStatus(status).
		Build()
	if err := tx.Commit(); err != nil {
		t.Fatalf("エクスポートの commit に失敗: %v", err)
	}
	return id
}

func countExports(t *testing.T, owner committedExportOwner) int {
	t.Helper()

	var count int
	if err := owner.db.QueryRow(
		"SELECT COUNT(*) FROM exports WHERE profile_id = $1",
		uuid.UUID(owner.ProfileID),
	).Scan(&count); err != nil {
		t.Fatalf("エクスポート件数の取得に失敗: %v", err)
	}
	return count
}

func exportExists(t *testing.T, owner committedExportOwner, id model.ExportID) bool {
	t.Helper()

	var exists bool
	if err := owner.db.QueryRow(
		"SELECT EXISTS (SELECT 1 FROM exports WHERE id = $1)",
		uuid.UUID(id),
	).Scan(&exists); err != nil {
		t.Fatalf("エクスポートの存在確認に失敗: %v", err)
	}
	return exists
}

func requireAppError(t *testing.T, err error, want model.AppErrorCode) {
	t.Helper()

	var ae *model.AppError
	if !errors.As(err, &ae) {
		t.Fatalf("*model.AppError を期待したが: %v", err)
	}
	if ae.Code != want {
		t.Fatalf("AppError.Code = %v, want %v", ae.Code, want)
	}
	if ae.UserMsg == "" {
		t.Error("AppError.UserMsg が空")
	}
}

func TestCreateExportUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), "ja")

	t.Run("queued のエクスポートを作成し生成ジョブを投入する", func(t *testing.T) {
		t.Parallel()

		owner := newCommittedExportOwner(t)
		inserter := &countingJobInserter{}
		uc := newCreateExportUsecase(owner.db, inserter, true)

		output, err := uc.Execute(ctx, usecase.CreateExportInput{
			UserID:    owner.UserID,
			ProfileID: owner.ProfileID,
			ActorID:   owner.ActorID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if output.Export == nil {
			t.Fatal("Export が返されていない")
		}
		if output.Export.Status != model.ExportStatusQueued {
			t.Errorf("Export.Status = %v, want %v", output.Export.Status, model.ExportStatusQueued)
		}
		if output.Export.ProfileID != owner.ProfileID {
			t.Errorf("Export.ProfileID = %v, want %v", output.Export.ProfileID, owner.ProfileID)
		}
		if output.Export.ActorID != owner.ActorID {
			t.Errorf("Export.ActorID = %v, want %v", output.Export.ActorID, owner.ActorID)
		}

		if !exportExists(t, owner, output.Export.ID) {
			t.Error("commit されたエクスポートが DB に無い")
		}

		got := inserter.generateExportIDs()
		if len(got) != 1 || got[0] != output.Export.ID.String() {
			t.Errorf("投入された生成ジョブ = %v, want [%s]", got, output.Export.ID)
		}
	})

	// The retention bound is one success plus one export that is in progress or
	// failed, so the request that replaces a failed attempt takes its slot in
	// the same commit.
	//
	// [Ja] 保持の上限は成功 1 件と、進行中または failed のエクスポート 1 件である。
	// failed な試行を置き換える申請は、同じ commit でその枠を引き継ぐ。
	t.Run("旧 failed を削除して新しい queued に置き換える", func(t *testing.T) {
		t.Parallel()

		owner := newCommittedExportOwner(t)
		failedID := commitExport(t, owner, model.ExportStatusFailed)
		succeededID := commitExport(t, owner, model.ExportStatusSucceeded)
		uc := newCreateExportUsecase(owner.db, &countingJobInserter{}, true)

		output, err := uc.Execute(ctx, usecase.CreateExportInput{
			UserID:    owner.UserID,
			ProfileID: owner.ProfileID,
			ActorID:   owner.ActorID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if exportExists(t, owner, failedID) {
			t.Error("旧 failed のエクスポートが残っている")
		}
		if !exportExists(t, owner, succeededID) {
			t.Error("成功したエクスポートが削除されている")
		}
		if !exportExists(t, owner, output.Export.ID) {
			t.Error("新しい queued のエクスポートが DB に無い")
		}
	})

	// Refusing before any write is what keeps a deployment without the object
	// storage from accumulating queued rows no Worker is registered to generate.
	//
	// [Ja] 何も書き込む前に拒否することが、オブジェクトストレージの無いデプロイで、
	// 生成する Worker の登録されていない queued の行が溜まるのを防ぐ。
	t.Run("ストレージ未設定なら 503 相当で拒否し行を作らない", func(t *testing.T) {
		t.Parallel()

		owner := newCommittedExportOwner(t)
		inserter := &countingJobInserter{}
		uc := newCreateExportUsecase(owner.db, inserter, false)

		output, err := uc.Execute(ctx, usecase.CreateExportInput{
			UserID:    owner.UserID,
			ProfileID: owner.ProfileID,
			ActorID:   owner.ActorID,
		})
		if output != nil {
			t.Error("Execute() output が返っている, want nil")
		}
		requireAppError(t, err, model.AppErrCodeServiceUnavailable)

		if n := countExports(t, owner); n != 0 {
			t.Errorf("エクスポートが %d 件作られている, want 0", n)
		}
		if got := inserter.generateExportIDs(); len(got) != 0 {
			t.Errorf("生成ジョブが投入されている: %v", got)
		}
	})

	// A profile the user does not own is refused as not found, so the response
	// cannot be used to tell an existing profile from a missing one.
	//
	// [Ja] ユーザーが所有していないプロフィールは not found として拒否する。応答から
	// 既存のプロフィールと存在しないプロフィールを区別できないようにするため。
	t.Run("他ユーザーのプロフィールは not found で拒否する", func(t *testing.T) {
		t.Parallel()

		owner := newCommittedExportOwner(t)
		other := newCommittedExportOwner(t)
		uc := newCreateExportUsecase(owner.db, &countingJobInserter{}, true)

		_, err := uc.Execute(ctx, usecase.CreateExportInput{
			UserID:    other.UserID,
			ProfileID: owner.ProfileID,
			ActorID:   owner.ActorID,
		})
		requireAppError(t, err, model.AppErrCodeResourceNotFound)

		if n := countExports(t, owner); n != 0 {
			t.Errorf("エクスポートが %d 件作られている, want 0", n)
		}
	})

	// Losing the race for the profile's one active slot must leave the failed
	// row standing: it is only obsolete once a replacement exists.
	//
	// [Ja] プロフィールの 1 つしかない実行枠を取り損ねた場合、failed の行は残さな
	// ければならない。置き換えるものができて初めて不要になるためである。
	for _, status := range []model.ExportStatus{model.ExportStatusQueued, model.ExportStatusStarted} {
		t.Run(status.String()+" のエクスポートがあれば競合として拒否する", func(t *testing.T) {
			t.Parallel()

			owner := newCommittedExportOwner(t)
			failedID := commitExport(t, owner, model.ExportStatusFailed)
			activeID := commitExport(t, owner, status)
			inserter := &countingJobInserter{}
			uc := newCreateExportUsecase(owner.db, inserter, true)

			_, err := uc.Execute(ctx, usecase.CreateExportInput{
				UserID:    owner.UserID,
				ProfileID: owner.ProfileID,
				ActorID:   owner.ActorID,
			})
			requireAppError(t, err, model.AppErrCodeConflict)

			if !exportExists(t, owner, failedID) {
				t.Error("競合で rollback したのに旧 failed が削除されている")
			}
			if !exportExists(t, owner, activeID) {
				t.Error("進行中のエクスポートが失われている")
			}
			if n := countExports(t, owner); n != 2 {
				t.Errorf("エクスポートが %d 件ある, want 2", n)
			}
			if got := inserter.generateExportIDs(); len(got) != 0 {
				t.Errorf("生成ジョブが投入されている: %v", got)
			}
		})
	}

	t.Run("同時に開始しても片方だけが成功する", func(t *testing.T) {
		t.Parallel()

		owner := newCommittedExportOwner(t)

		var wg sync.WaitGroup
		results := make([]error, 2)
		for i := range results {
			wg.Add(1)
			go func() {
				defer wg.Done()
				uc := newCreateExportUsecase(owner.db, &countingJobInserter{}, true)
				_, err := uc.Execute(ctx, usecase.CreateExportInput{
					UserID:    owner.UserID,
					ProfileID: owner.ProfileID,
					ActorID:   owner.ActorID,
				})
				results[i] = err
			}()
		}
		wg.Wait()

		var succeeded, conflicted int
		for _, err := range results {
			switch {
			case err == nil:
				succeeded++
			case model.AsAppError(err) != nil && model.AsAppError(err).Code == model.AppErrCodeConflict:
				conflicted++
			default:
				t.Errorf("想定外のエラー: %v", err)
			}
		}
		if succeeded != 1 || conflicted != 1 {
			t.Errorf("成功 %d 件 / 競合 %d 件, want 1 件 / 1 件", succeeded, conflicted)
		}

		if n := countExports(t, owner); n != 1 {
			t.Errorf("エクスポートが %d 件ある, want 1", n)
		}
	})

	// The queued row is the durable work intent, and reconciliation inserts a
	// job for a queued export that never got one, so a failed insert must not
	// deny an export that is going to run.
	//
	// [Ja] queued の行が durable work intent であり、ジョブを得られなかった queued の
	// エクスポートにはリコンシリエーションがジョブを投入する。投入の失敗が、これから
	// 実行されるエクスポートを拒否してはならない。
	t.Run("ジョブ投入に失敗しても queued を残して成功を返す", func(t *testing.T) {
		t.Parallel()

		owner := newCommittedExportOwner(t)
		inserter := &countingJobInserter{err: errors.New("ジョブキューに接続できない")}
		uc := newCreateExportUsecase(owner.db, inserter, true)

		output, err := uc.Execute(ctx, usecase.CreateExportInput{
			UserID:    owner.UserID,
			ProfileID: owner.ProfileID,
			ActorID:   owner.ActorID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v, want nil", err)
		}

		if !exportExists(t, owner, output.Export.ID) {
			t.Error("投入失敗後に queued のエクスポートが残っていない")
		}
		if output.Export.Status != model.ExportStatusQueued {
			t.Errorf("Export.Status = %v, want %v", output.Export.Status, model.ExportStatusQueued)
		}
	})

	// The posts an export would archive are on their way out, so the request is
	// refused instead of creating a row profile cleanup would have to chase.
	//
	// [Ja] エクスポートがアーカイブするはずのポストは、これから消えていくもので
	// ある。プロフィールの cleanup が追いかける羽目になる行を作らず、申請を拒否する。
	t.Run("プロフィール削除が始まっていれば競合として拒否する", func(t *testing.T) {
		t.Parallel()

		owner := newCommittedExportOwner(t)
		if _, err := owner.db.Exec(
			"UPDATE profiles SET export_deletion_started_at = NOW() WHERE id = $1",
			uuid.UUID(owner.ProfileID),
		); err != nil {
			t.Fatalf("プロフィール削除開始の記録に失敗: %v", err)
		}

		inserter := &countingJobInserter{}
		uc := newCreateExportUsecase(owner.db, inserter, true)

		_, err := uc.Execute(ctx, usecase.CreateExportInput{
			UserID:    owner.UserID,
			ProfileID: owner.ProfileID,
			ActorID:   owner.ActorID,
		})
		requireAppError(t, err, model.AppErrCodeConflict)

		if n := countExports(t, owner); n != 0 {
			t.Errorf("エクスポートが %d 件作られている, want 0", n)
		}
		if got := inserter.generateExportIDs(); len(got) != 0 {
			t.Errorf("生成ジョブが投入されている: %v", got)
		}
	})
}
