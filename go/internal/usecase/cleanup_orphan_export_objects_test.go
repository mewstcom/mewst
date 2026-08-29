package usecase_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// storedObject is one object the sweep's listing hands over.
//
// [Ja] storedObject は掃除の一覧取得が渡すオブジェクト 1 件。
type storedObject struct {
	key          string
	lastModified time.Time
}

// sweepObjectStorage stands in for the object storage of the orphan sweep. It
// lists the objects a test placed in it, records the deletions, and can fail a
// chosen key or the listing itself so the sweep can be observed in the states a
// partly unavailable storage produces.
//
// [Ja] sweepObjectStorage は孤児回収におけるオブジェクトストレージの代役。テストが
// 置いたオブジェクトを一覧し、削除を記録する。指定したキーや一覧取得そのものを失敗
// させられるため、一部が利用できないストレージで生じる状態を観測できる。
type sweepObjectStorage struct {
	t *testing.T

	objects    []storedObject
	deleteErrs map[string]error
	listErr    error

	// beforeYield runs before each object is handed over, letting a test break
	// something the sweep depends on while the walk is still in progress.
	//
	// [Ja] beforeYield は各オブジェクトを渡す前に実行される。走査の途中で掃除が依存
	// するものをテストが壊せるようにする。
	beforeYield func()

	mu          sync.Mutex
	deleted     []string
	yielded     int
	startAfters []string
}

func newSweepObjectStorage(t *testing.T) *sweepObjectStorage {
	t.Helper()
	return &sweepObjectStorage{t: t, deleteErrs: map[string]error{}}
}

// put adds an object to the listing.
//
// [Ja] put は一覧されるオブジェクトを追加する。
func (s *sweepObjectStorage) put(key string, lastModified time.Time) string {
	s.objects = append(s.objects, storedObject{key: key, lastModified: lastModified})
	return key
}

func (s *sweepObjectStorage) Upload(_ context.Context, key string, _ io.Reader) error {
	s.t.Errorf("孤児回収は Upload を呼ばないはず (key: %s)", key)
	return nil
}

func (s *sweepObjectStorage) Download(_ context.Context, key string) (io.ReadCloser, int64, error) {
	s.t.Errorf("孤児回収は Download を呼ばないはず (key: %s)", key)
	return nil, 0, nil
}

func (s *sweepObjectStorage) Delete(_ context.Context, key string) error {
	if err := s.deleteErrs[key]; err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleted = append(s.deleted, key)
	return nil
}

func (s *sweepObjectStorage) ListPrefix(_ context.Context, prefix, startAfter string, yield func(key string, lastModified time.Time) error) error {
	// Listing outside the export prefix would enumerate the objects other
	// features store in the same bucket.
	//
	// [Ja] エクスポートのプレフィックスの外を一覧すると、同じバケットに他機能が
	// 保存したオブジェクトまで列挙することになる。
	if prefix != usecase.ExportObjectKeyPrefix {
		s.t.Errorf("一覧取得の prefix = %q, want %q", prefix, usecase.ExportObjectKeyPrefix)
	}
	if s.listErr != nil {
		return s.listErr
	}

	s.mu.Lock()
	s.startAfters = append(s.startAfters, startAfter)
	s.mu.Unlock()

	// The listing hands the objects over in key order, as the storage does. The
	// resume position only means anything against that order, and so does the
	// key the sweep hands back after stopping partway.
	//
	// [Ja] 一覧はストレージと同じくキー順にオブジェクトを渡す。再開位置が意味を持つのは
	// この並びに対してだけであり、掃除が途中で止まって返すキーも同じである。
	objects := slices.SortedFunc(slices.Values(s.objects), func(a, b storedObject) int {
		return strings.Compare(a.key, b.key)
	})

	for _, object := range objects {
		if startAfter != "" && object.key <= startAfter {
			continue
		}

		if s.beforeYield != nil {
			s.beforeYield()
		}

		s.mu.Lock()
		s.yielded++
		s.mu.Unlock()

		if err := yield(object.key, object.lastModified); err != nil {
			// The S3 adapter wraps a callback failure in its own listing message.
			// Mirroring that here keeps the fake from making a failure easier to
			// classify than it is in production.
			//
			// [Ja] S3 アダプタはコールバックの失敗を自身の一覧取得のメッセージで
			// ラップする。ここでも同じ形にすることで、この代役が本番より失敗を
			// 分類しやすくしてしまわないようにする。
			return fmt.Errorf("オブジェクト一覧の処理に失敗 (prefix: %s): %w", prefix, err)
		}
	}
	return nil
}

func (s *sweepObjectStorage) deletedKeys() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.deleted)
}

// yieldedCount returns how many objects the listing handed over, which tells a
// test whether the walk ran to the end or stopped partway.
//
// [Ja] yieldedCount は一覧取得が渡したオブジェクト数を返す。走査が最後まで進んだか
// 途中で止まったかをテストが判別できる。
func (s *sweepObjectStorage) yieldedCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.yielded
}

// requestedStartAfters returns the resume position of each listing the sweep
// asked for.
//
// [Ja] requestedStartAfters は、掃除が要求した各一覧取得の再開位置を返す。
func (s *sweepObjectStorage) requestedStartAfters() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.startAfters)
}

// newExportObjectKey creates an export with the given status and returns the
// object key its archive would be stored under.
//
// [Ja] newExportObjectKey は指定 status のエクスポートを作成し、そのアーカイブが
// 保存されるオブジェクトキーを返す。
func newExportObjectKey(t *testing.T, tx *sql.Tx, status model.ExportStatus) string {
	t.Helper()

	target := testutil.NewProfileOwner(t, tx)
	exportID := testutil.NewExportBuilder(t, tx).
		WithProfileID(target.ProfileID).
		WithActorID(target.ActorID).
		WithStatus(status).
		Build()
	return usecase.ExportObjectKey(target.ProfileID, exportID)
}

// newOrphanObjectKey returns the key of an object whose export has no row at
// all, which is what a deleted export leaves behind.
//
// [Ja] newOrphanObjectKey は、エクスポートの行がまったく存在しないオブジェクトの
// キーを返す。削除済みのエクスポートが残すのがこの状態である。
func newOrphanObjectKey() string {
	return usecase.ExportObjectKey(model.ProfileID(uuid.New()), model.ExportID(uuid.New()))
}

func newCleanupOrphanExportObjectsUsecase(
	t *testing.T,
	tx *sql.Tx,
	storage *sweepObjectStorage,
	inserter *exportJobInserter,
	limits usecase.ExportOrphanSweepLimits,
) *usecase.CleanupOrphanExportObjectsUsecase {
	t.Helper()

	exportRepo := repository.NewExportRepository(testutil.QueriesWithTx(tx))
	return usecase.NewCleanupOrphanExportObjectsUsecase(
		exportRepo,
		storage,
		dispatcher.NewDispatcher(inserter),
		limits,
	)
}

// TestCleanupOrphanExportObjectsUsecase_Execute pins which stored objects the
// sweep is allowed to delete. Deleting an object an export still retains would
// take away an archive a user can download, and keeping one nothing retains is
// the storage cost this sweep exists to end.
//
// [Ja] TestCleanupOrphanExportObjectsUsecase_Execute は、掃除が削除してよい保存済み
// オブジェクトを固定する。エクスポートがまだ保持しているオブジェクトを削除すると、
// ユーザーがダウンロードできるアーカイブを奪うことになり、何も保持していない
// オブジェクトを残すことは、この掃除が終わらせるために存在するストレージのコストになる。
func TestCleanupOrphanExportObjectsUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	old := time.Now().Add(-48 * time.Hour)

	t.Run("保持しているエクスポートが無いオブジェクトだけを削除する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		storage := newSweepObjectStorage(t)

		// The archive currently offered for download.
		//
		// [Ja] 現在ダウンロード対象になっているアーカイブ。
		succeeded := storage.put(newExportObjectKey(t, tx, model.ExportStatusSucceeded), old)
		// An attempt that has not run yet and one that is running: a retry
		// overwrites this key, so the object belongs to work still in progress.
		//
		// [Ja] まだ実行されていない試行と実行中の試行。リトライがこのキーを上書き
		// するため、オブジェクトは進行中の処理に属している。
		queued := storage.put(newExportObjectKey(t, tx, model.ExportStatusQueued), old)
		started := storage.put(newExportObjectKey(t, tx, model.ExportStatusStarted), old)
		// A terminal failure releases its object. This is the window between the
		// failed transition and the deletion that follows it, which is exactly what
		// the sweep is here to close.
		//
		// [Ja] 終端の失敗はオブジェクトを手放す。これは failed への遷移とその後の削除の
		// 間の窓であり、掃除が閉じるべきものそのものである。
		failed := storage.put(newExportObjectKey(t, tx, model.ExportStatusFailed), old)
		// No row at all: the export was deleted after its object was left behind.
		//
		// [Ja] 行がまったく無い状態。オブジェクトを残したままエクスポートが削除された
		// 場合。
		orphan := storage.put(newOrphanObjectKey(), old)

		uc := newCleanupOrphanExportObjectsUsecase(t, tx, storage, newExportJobInserter(t), usecase.DefaultExportOrphanSweepLimits())
		if err := uc.Execute(ctx, ""); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		got := storage.deletedKeys()
		slices.Sort(got)
		want := []string{failed, orphan}
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Errorf("削除されたキー = %v, want %v", got, want)
		}
		for _, key := range []string{succeeded, queued, started} {
			if slices.Contains(got, key) {
				t.Errorf("保持中のエクスポートのオブジェクト %s を削除しました", key)
			}
		}
	})

	t.Run("猶予期間内に書き込まれたオブジェクトには触れない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		storage := newSweepObjectStorage(t)

		// An upload finishes before the transition that records it, and the
		// listing happens before the matching query, so a just-written object can
		// legitimately have no row retaining it yet.
		//
		// [Ja] アップロードはそれを記録する遷移より先に完了し、一覧取得は照合クエリより
		// 先に行われる。そのため、書き込まれたばかりのオブジェクトに、まだそれを保持する
		// 行が無いことは正当にありうる。
		justWritten := storage.put(newOrphanObjectKey(), time.Now())

		uc := newCleanupOrphanExportObjectsUsecase(t, tx, storage, newExportJobInserter(t), usecase.DefaultExportOrphanSweepLimits())
		if err := uc.Execute(ctx, ""); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if slices.Contains(storage.deletedKeys(), justWritten) {
			t.Errorf("猶予期間内のオブジェクト %s を削除しました", justWritten)
		}
	})

	t.Run("規約に従わないキーは削除せず走査を続ける", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		storage := newSweepObjectStorage(t)

		// A key the export convention cannot explain names no export, so the sweep
		// has no way to tell whether anything still needs it.
		//
		// [Ja] エクスポートの規約で説明できないキーはどのエクスポートも指さないため、
		// 掃除には、それをまだ必要としているものがあるかどうかを判断する手段が無い。
		foreign := storage.put(usecase.ExportObjectKeyPrefix+"not-an-export/README.txt", old)
		orphan := storage.put(newOrphanObjectKey(), old)

		uc := newCleanupOrphanExportObjectsUsecase(t, tx, storage, newExportJobInserter(t), usecase.DefaultExportOrphanSweepLimits())
		if err := uc.Execute(ctx, ""); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		got := storage.deletedKeys()
		if slices.Contains(got, foreign) {
			t.Errorf("規約外のキー %s を削除しました", foreign)
		}
		if !slices.Contains(got, orphan) {
			t.Errorf("孤児オブジェクト %s が削除されていません", orphan)
		}
	})

	t.Run("バッチ境界をまたいでも全ての孤児オブジェクトを回収する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		storage := newSweepObjectStorage(t)

		// More objects than one match fits, so the listing spans several batches
		// and the last, partly filled one has to be matched as well.
		//
		// [Ja] 1 回の照合に収まらない数のオブジェクトを置き、一覧が複数バッチにまたがる
		// ようにする。最後の埋まりきっていないバッチも照合される必要がある。
		const objectCount = 250
		orphans := make([]string, 0, objectCount)
		for range objectCount {
			orphans = append(orphans, storage.put(newOrphanObjectKey(), old))
		}

		uc := newCleanupOrphanExportObjectsUsecase(t, tx, storage, newExportJobInserter(t), usecase.DefaultExportOrphanSweepLimits())
		if err := uc.Execute(ctx, ""); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		got := storage.deletedKeys()
		slices.Sort(got)
		slices.Sort(orphans)
		if !slices.Equal(got, orphans) {
			t.Errorf("削除されたキーの件数 = %d, want %d", len(got), len(orphans))
		}
	})

	t.Run("1 件の削除失敗が後続のオブジェクトの回収を止めない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		storage := newSweepObjectStorage(t)

		refused := storage.put(newOrphanObjectKey(), old)
		behind := storage.put(newOrphanObjectKey(), old)
		storage.deleteErrs[refused] = errors.New("注入したストレージのエラー")

		uc := newCleanupOrphanExportObjectsUsecase(t, tx, storage, newExportJobInserter(t), usecase.DefaultExportOrphanSweepLimits())
		err := uc.Execute(ctx, "")
		if err == nil {
			t.Fatal("Execute() = nil, want an error")
		}

		// The refused object stays for the next run, but the ones behind it are
		// collected now: a single object the storage will not part with must not
		// hold the whole sweep hostage.
		//
		// [Ja] 拒否されたオブジェクトは次回の実行に残るが、その後ろにあるものは今回
		// 回収される。ストレージが手放さない 1 件のオブジェクトが掃除全体を人質に
		// 取ってはならない。
		if !slices.Contains(storage.deletedKeys(), behind) {
			t.Errorf("後続の孤児オブジェクト %s が削除されていません", behind)
		}
	})

	// A failed match and a failed listing name different systems: the database
	// and the object storage. The match runs from inside the listing, so its
	// failure travels back wrapped in the listing's own message and would be
	// reported as a storage failure unless the sweep keeps the two apart.
	//
	// [Ja] 照合の失敗と一覧取得の失敗は別のシステムを指す。DB とオブジェクトストレージ
	// である。照合は一覧取得の内側から走るため、その失敗は一覧取得自身のメッセージに
	// ラップされて戻り、掃除が両者を分けて保持しない限りストレージの障害として報告されて
	// しまう。
	t.Run("照合の失敗を一覧取得の失敗として報告しない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		storage := newSweepObjectStorage(t)

		// More objects than one batch holds, so the match runs while the walk is
		// still in progress.
		//
		// [Ja] 1 バッチに収まらない数のオブジェクトを置き、走査の途中で照合が走るように
		// する。
		const objectCount = 250
		for range objectCount {
			storage.put(newOrphanObjectKey(), old)
		}

		// Ending the transaction is what a database the sweep can no longer reach
		// looks like from inside the walk.
		//
		// [Ja] トランザクションを終わらせることが、走査の内側から見て「掃除が到達でき
		// なくなった DB」にあたる。
		storage.beforeYield = func() { _ = tx.Rollback() }

		uc := newCleanupOrphanExportObjectsUsecase(t, tx, storage, newExportJobInserter(t), usecase.DefaultExportOrphanSweepLimits())
		err := uc.Execute(ctx, "")
		if err == nil {
			t.Fatal("Execute() = nil, want an error")
		}
		if !strings.Contains(err.Error(), "オブジェクトを保持するエクスポートの照合に失敗") {
			t.Errorf("Execute() error = %v, want it to report the match failure", err)
		}
		if strings.Contains(err.Error(), "一覧取得に失敗") {
			t.Errorf("Execute() error = %v, want it not to be reported as a listing failure", err)
		}
		if got := storage.yieldedCount(); got >= objectCount {
			t.Errorf("一覧取得が渡した件数 = %d、照合の失敗で走査が止まっていません", got)
		}
	})

	t.Run("一覧取得の失敗はエラーとして返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		storage := newSweepObjectStorage(t)
		storage.listErr = fmt.Errorf("注入した一覧取得のエラー")

		uc := newCleanupOrphanExportObjectsUsecase(t, tx, storage, newExportJobInserter(t), usecase.DefaultExportOrphanSweepLimits())
		if err := uc.Execute(ctx, ""); err == nil {
			t.Fatal("Execute() = nil, want an error")
		}
		if got := storage.deletedKeys(); len(got) != 0 {
			t.Errorf("一覧取得に失敗した実行が %v を削除しました", got)
		}
	})
}

// TestCleanupOrphanExportObjectsUsecase_Continuation pins how a prefix larger
// than one run is covered. A run bounded only by its timeout would be retried
// from the same position and keep stopping at the same place, so the objects
// past that point would never be reached. Handing the position over is what
// makes the walk advance across runs.
//
// [Ja] TestCleanupOrphanExportObjectsUsecase_Continuation は、1 回の実行に収まらない
// プレフィックスをどう網羅するかを固定する。timeout だけで区切られた実行は同じ位置から
// 再試行され同じ場所で止まり続けるため、その先のオブジェクトには決して到達しない。位置を
// 引き渡すことが、走査を実行またぎで前進させる。
func TestCleanupOrphanExportObjectsUsecase_Continuation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	old := time.Now().Add(-48 * time.Hour)

	// One object per run, so the second object is only reachable through a
	// continuation.
	//
	// [Ja] 1 回の実行につき 1 オブジェクトとし、2 件目には継続ジョブを通してしか到達
	// できないようにする。
	oneObjectPerRun := usecase.ExportOrphanSweepLimits{BatchSize: 1, ScanBudget: 1}

	t.Run("走査予算を使い切ったら止まった位置から継続ジョブを投入する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		storage := newSweepObjectStorage(t)
		inserter := newExportJobInserter(t)

		keys := []string{newOrphanObjectKey(), newOrphanObjectKey()}
		slices.Sort(keys)
		for _, key := range keys {
			storage.put(key, old)
		}

		uc := newCleanupOrphanExportObjectsUsecase(t, tx, storage, inserter, oneObjectPerRun)
		if err := uc.Execute(ctx, ""); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		// The first object is collected now and the second is left to the job the
		// run enqueued, which resumes after the key the walk stopped at.
		//
		// [Ja] 1 件目は今回回収し、2 件目はこの実行が投入したジョブに委ねる。そのジョブは
		// 走査が止まったキーの次から再開する。
		if got := storage.deletedKeys(); !slices.Equal(got, keys[:1]) {
			t.Errorf("削除されたキー = %v, want %v", got, keys[:1])
		}
		assertJobFor(t, inserter, "cleanup_orphan_export_objects", keys[0])
	})

	t.Run("継続ジョブは引き渡された位置の次から走査する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		storage := newSweepObjectStorage(t)
		inserter := newExportJobInserter(t)

		keys := []string{newOrphanObjectKey(), newOrphanObjectKey()}
		slices.Sort(keys)
		for _, key := range keys {
			storage.put(key, old)
		}

		uc := newCleanupOrphanExportObjectsUsecase(t, tx, storage, inserter, oneObjectPerRun)
		if err := uc.Execute(ctx, keys[0]); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if got := storage.requestedStartAfters(); !slices.Equal(got, []string{keys[0]}) {
			t.Errorf("一覧取得の再開位置 = %v, want %v", got, []string{keys[0]})
		}
		if got := storage.deletedKeys(); !slices.Equal(got, keys[1:]) {
			t.Errorf("削除されたキー = %v, want %v", got, keys[1:])
		}
	})

	// The walk ending inside the budget means there is nothing left, so a
	// continuation would insert a job that lists an empty remainder every day.
	//
	// [Ja] 予算の内側で走査が終わったということは残りが無いということであり、継続ジョブを
	// 投入すると、毎日空の残りを一覧するだけのジョブを投入することになる。
	t.Run("走査し終えたら継続ジョブを投入しない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		storage := newSweepObjectStorage(t)
		inserter := newExportJobInserter(t)
		storage.put(newOrphanObjectKey(), old)

		uc := newCleanupOrphanExportObjectsUsecase(t, tx, storage, inserter, usecase.DefaultExportOrphanSweepLimits())
		if err := uc.Execute(ctx, ""); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if got := inserter.attemptedIDs("cleanup_orphan_export_objects"); len(got) != 0 {
			t.Errorf("継続ジョブが %v に対して投入されました", got)
		}
	})

	// A failed run is retried by the job queue from the same position. Handing
	// the rest over as well would have the retry and the continuation walk the
	// same objects.
	//
	// [Ja] 失敗した実行はジョブキューが同じ位置から再試行する。ここで残りも引き渡すと、
	// 再試行と継続ジョブが同じオブジェクトを走査することになる。
	t.Run("削除に失敗した実行は継続ジョブを投入しない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		storage := newSweepObjectStorage(t)
		inserter := newExportJobInserter(t)

		keys := []string{newOrphanObjectKey(), newOrphanObjectKey()}
		slices.Sort(keys)
		for _, key := range keys {
			storage.put(key, old)
		}
		// The refused object is the first the walk reaches, so the run ends on a
		// failure with a remainder still unwalked.
		//
		// [Ja] 拒否されるオブジェクトを走査が最初に到達する位置に置き、残りを走査しない
		// まま実行が失敗で終わるようにする。
		storage.deleteErrs[keys[0]] = errors.New("注入したストレージのエラー")

		uc := newCleanupOrphanExportObjectsUsecase(t, tx, storage, inserter, oneObjectPerRun)
		if err := uc.Execute(ctx, ""); err == nil {
			t.Fatal("Execute() = nil, want an error")
		}

		if got := inserter.attemptedIDs("cleanup_orphan_export_objects"); len(got) != 0 {
			t.Errorf("失敗した実行が継続ジョブを %v に対して投入しました", got)
		}
	})
}
