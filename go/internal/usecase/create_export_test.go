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

// countingJobInserterは投入を依頼されたジョブを記録し、失敗させることも
// できる。recordingJobInserterと違い状態をmutexで守るのは、同時Createの
// テストが2つのExecuteを同時に走らせるためである。
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

// newCreateExportUsecaseは共有テストDBに対して動くCreateExportUsecaseを
// 構築する。このUseCaseは自身のtransactionを開くため、前提となる行はアウターの
// transactionに保持したままではなくcommitしておく必要がある。
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

// committedExportOwnerはプロフィールを所有するユーザーで、UseCase自身の
// transactionから見えるよう共有テストDBへcommitしてある。
type committedExportOwner struct {
	testutil.ProfileOwner
	db *sql.DB
}

func newCommittedExportOwner(t *testing.T) committedExportOwner {
	t.Helper()

	db := testutil.GetTestDB()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("前提データ用transactionの開始に失敗: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	owner := testutil.NewProfileOwner(t, tx)
	if err := tx.Commit(); err != nil {
		t.Fatalf("前提データのcommitに失敗: %v", err)
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

// commitExportは指定statusのエクスポートを1件その所有者に作ってcommit
// する。UseCaseのtransactionが既存の状態として観測できるようにするため。
func commitExport(t *testing.T, owner committedExportOwner, status model.ExportStatus) model.ExportID {
	t.Helper()

	tx, err := owner.db.Begin()
	if err != nil {
		t.Fatalf("エクスポート作成用transactionの開始に失敗: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	id := testutil.NewExportBuilder(t, tx).
		WithProfileID(owner.ProfileID).
		WithActorID(owner.ActorID).
		WithStatus(status).
		Build()
	if err := tx.Commit(); err != nil {
		t.Fatalf("エクスポートのcommitに失敗: %v", err)
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
		t.Fatalf("*model.AppErrorを期待したが: %v", err)
	}
	if ae.Code != want {
		t.Fatalf("AppError.Code = %v、期待値 = %v", ae.Code, want)
	}
	if ae.UserMsg == "" {
		t.Error("AppError.UserMsgが空")
	}
}

func TestCreateExportUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), "ja")

	t.Run("queuedのエクスポートを作成し生成ジョブを投入する", func(t *testing.T) {
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
			t.Fatalf("Execute()のエラー = %v", err)
		}

		if output.Export == nil {
			t.Fatal("Exportが返されていない")
		}
		if output.Export.Status != model.ExportStatusQueued {
			t.Errorf("Export.Status = %v、期待値 = %v", output.Export.Status, model.ExportStatusQueued)
		}
		if output.Export.ProfileID != owner.ProfileID {
			t.Errorf("Export.ProfileID = %v、期待値 = %v", output.Export.ProfileID, owner.ProfileID)
		}
		if output.Export.ActorID != owner.ActorID {
			t.Errorf("Export.ActorID = %v、期待値 = %v", output.Export.ActorID, owner.ActorID)
		}

		if !exportExists(t, owner, output.Export.ID) {
			t.Error("commitされたエクスポートがDBに無い")
		}

		got := inserter.generateExportIDs()
		if len(got) != 1 || got[0] != output.Export.ID.String() {
			t.Errorf("投入された生成ジョブ = %v、期待値 = [%s]", got, output.Export.ID)
		}
	})

	// 保持の上限は成功1件と、進行中またはfailedのエクスポート1件である。
	// failedな試行を置き換える申請は、同じcommitでその枠を引き継ぐ。
	t.Run("旧failedを削除して新しいqueuedに置き換える", func(t *testing.T) {
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
			t.Fatalf("Execute()のエラー = %v", err)
		}

		if exportExists(t, owner, failedID) {
			t.Error("旧failedのエクスポートが残っている")
		}
		if !exportExists(t, owner, succeededID) {
			t.Error("成功したエクスポートが削除されている")
		}
		if !exportExists(t, owner, output.Export.ID) {
			t.Error("新しいqueuedのエクスポートがDBに無い")
		}
	})

	// 何も書き込む前に拒否することが、オブジェクトストレージの無いデプロイで、
	// 生成するWorkerの登録されていないqueuedの行が溜まるのを防ぐ。
	t.Run("ストレージ未設定なら503相当で拒否し行を作らない", func(t *testing.T) {
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
			t.Error("Execute()の出力 = 非nil、期待値 = nil")
		}
		requireAppError(t, err, model.AppErrCodeServiceUnavailable)

		if n := countExports(t, owner); n != 0 {
			t.Errorf("エクスポートが%d件作られている、期待値 = 0", n)
		}
		if got := inserter.generateExportIDs(); len(got) != 0 {
			t.Errorf("生成ジョブが投入されている: %v", got)
		}
	})

	// ユーザーが所有していないプロフィールはnot foundとして拒否する。応答から
	// 既存のプロフィールと存在しないプロフィールを区別できないようにするため。
	t.Run("他ユーザーのプロフィールはnot foundで拒否する", func(t *testing.T) {
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
			t.Errorf("エクスポートが%d件作られている、期待値 = 0", n)
		}
	})

	// プロフィールの1つしかない実行枠を取り損ねた場合、failedの行は残さな
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
				t.Error("競合でrollbackしたのに旧failedが削除されている")
			}
			if !exportExists(t, owner, activeID) {
				t.Error("進行中のエクスポートが失われている")
			}
			if n := countExports(t, owner); n != 2 {
				t.Errorf("エクスポートが%d件ある、期待値 = 2", n)
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
			t.Errorf("成功%d件 / 競合%d件、期待値 = 1件 / 1件", succeeded, conflicted)
		}

		if n := countExports(t, owner); n != 1 {
			t.Errorf("エクスポートが%d件ある、期待値 = 1", n)
		}
	})

	// queuedの行がdurable work intentであり、ジョブを得られなかったqueuedの
	// エクスポートにはリコンシリエーションがジョブを投入する。投入の失敗が、これから
	// 実行されるエクスポートを拒否してはならない。
	t.Run("ジョブ投入に失敗してもqueuedを残して成功を返す", func(t *testing.T) {
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
			t.Fatalf("Execute()のエラー = %v、期待値 = nil", err)
		}

		if !exportExists(t, owner, output.Export.ID) {
			t.Error("投入失敗後にqueuedのエクスポートが残っていない")
		}
		if output.Export.Status != model.ExportStatusQueued {
			t.Errorf("Export.Status = %v、期待値 = %v", output.Export.Status, model.ExportStatusQueued)
		}
	})

	// エクスポートがアーカイブするはずのポストは、これから消えていくもので
	// ある。プロフィールのcleanupが追いかける羽目になる行を作らず、申請を拒否する。
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
			t.Errorf("エクスポートが%d件作られている、期待値 = 0", n)
		}
		if got := inserter.generateExportIDs(); len(got) != 0 {
			t.Errorf("生成ジョブが投入されている: %v", got)
		}
	})
}
