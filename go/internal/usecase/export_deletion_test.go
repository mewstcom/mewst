package usecase_test

import (
	"context"
	"database/sql"
	"io"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// exportDeletionPageSizeForTests is larger than any profile these tests build,
// so a listing never truncates a result the assertions want in full.
//
// [Ja] exportDeletionPageSizeForTests はこれらのテストが作るどのプロフィールより
// 大きく、検証が全件を見たい結果を一覧が切り詰めないようにする。
const exportDeletionPageSizeForTests int32 = 1000

// exportDeletionObjectStorage stands in for the object storage the export
// deletions call. It holds the objects a test placed in it, records what was
// deleted and from which key, and can refuse a chosen key so the deletion can
// be observed in the state a partly unavailable storage produces.
//
// [Ja] exportDeletionObjectStorage は、エクスポートの削除処理が呼ぶオブジェクト
// ストレージの代役。テストが置いたオブジェクトを保持し、何がどのキーから削除された
// かを記録する。指定したキーを拒否させられるため、一部が利用できないストレージで
// 生じる状態を観測できる。
type exportDeletionObjectStorage struct {
	t *testing.T

	// beforeDelete runs before each deletion, letting a test change the exports
	// table while the run is between two of its candidates.
	//
	// [Ja] beforeDelete は各削除の前に実行される。実行が候補と候補の間にいる間に、
	// テストが exports テーブルを変更できるようにする。
	beforeDelete func(key string)

	mu         sync.Mutex
	objects    map[string]struct{}
	deleteErrs map[string]error
	deleted    []string
}

func newExportDeletionObjectStorage(t *testing.T) *exportDeletionObjectStorage {
	t.Helper()
	return &exportDeletionObjectStorage{
		t:          t,
		objects:    map[string]struct{}{},
		deleteErrs: map[string]error{},
	}
}

// put stores an object under the key, standing for an archive an export
// uploaded.
//
// [Ja] put はキーの位置にオブジェクトを保存する。エクスポートがアップロードした
// アーカイブを表す。
func (s *exportDeletionObjectStorage) put(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects[key] = struct{}{}
}

// failOn makes the storage refuse the key, standing for an object the storage
// cannot remove right now.
//
// [Ja] failOn はストレージにそのキーを拒否させる。今は削除できないオブジェクトを
// 表す。
func (s *exportDeletionObjectStorage) failOn(key string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleteErrs[key] = err
}

// Delete removes the object. A key with no object is answered with success, as
// the S3 adapter answers a deletion of an object that is already gone, so a
// rerun of a deletion that stopped between the object and the row can finish.
//
// [Ja] Delete はオブジェクトを削除する。オブジェクトの無いキーには成功を返す。
// すでに存在しないオブジェクトの削除に S3 アダプタが返すのと同じであり、これに
// より、オブジェクトと行の間で止まった削除の再実行が完了できる。
func (s *exportDeletionObjectStorage) Delete(_ context.Context, key string) error {
	if s.beforeDelete != nil {
		s.beforeDelete(key)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.deleteErrs[key]; err != nil {
		return err
	}
	delete(s.objects, key)
	s.deleted = append(s.deleted, key)
	return nil
}

func (s *exportDeletionObjectStorage) Upload(_ context.Context, key string, _ io.Reader) error {
	s.t.Errorf("エクスポートの削除は Upload を呼ばないはず (key: %s)", key)
	return nil
}

func (s *exportDeletionObjectStorage) Download(_ context.Context, key string) (io.ReadCloser, int64, error) {
	s.t.Errorf("エクスポートの削除は Download を呼ばないはず (key: %s)", key)
	return nil, 0, nil
}

func (s *exportDeletionObjectStorage) ListPrefix(_ context.Context, prefix, _ string, _ func(key string, lastModified time.Time) error) error {
	// The deletions know which objects belong to their exports from the rows
	// themselves, so listing would enumerate objects they were never asked about.
	//
	// [Ja] 削除処理は自分のエクスポートのオブジェクトを行そのものから知るため、一覧
	// 取得は、依頼されていないオブジェクトまで列挙することになる。
	s.t.Errorf("エクスポートの削除は ListPrefix を呼ばないはず (prefix: %s)", prefix)
	return nil
}

// deletedKeys returns the keys whose object the run removed, in the order it
// removed them.
//
// [Ja] deletedKeys は実行がオブジェクトを削除したキーを、削除した順に返す。
func (s *exportDeletionObjectStorage) deletedKeys() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.deleted)
}

// storedKeys returns the keys that still hold an object.
//
// [Ja] storedKeys はまだオブジェクトを保持しているキーを返す。
func (s *exportDeletionObjectStorage) storedKeys() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	keys := make([]string, 0, len(s.objects))
	for key := range s.objects {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

// buildStoredExport creates an export of the given status for the target and
// stores its archive, returning the export ID and the key the archive is under.
// Both parts exist for every status because an attempt uploads before the
// transition that records it, so a queued, started or failed export can own an
// object as well.
//
// [Ja] buildStoredExport は対象に対して指定 status のエクスポートを作成し、その
// アーカイブを保存して、エクスポート ID とアーカイブのキーを返す。どの status でも
// 両方が存在するのは、試行がそれを記録する遷移より先にアップロードするためで、
// queued / started / failed のエクスポートもオブジェクトを持ちうる。
func buildStoredExport(
	t *testing.T,
	tx *sql.Tx,
	storage *exportDeletionObjectStorage,
	target testutil.ProfileOwner,
	status model.ExportStatus,
	createdAt time.Time,
) (model.ExportID, string) {
	t.Helper()

	exportID := testutil.NewExportBuilder(t, tx).
		WithProfileID(target.ProfileID).
		WithActorID(target.ActorID).
		WithStatus(status).
		WithCreatedAt(createdAt).
		Build()

	key := usecase.ExportObjectKey(target.ProfileID, exportID)
	storage.put(key)
	return exportID, key
}

// remainingExportIDs returns the IDs of the profile's exports that still have a
// row, oldest first.
//
// [Ja] remainingExportIDs はプロフィールのエクスポートのうち行が残っているものの
// ID を、古い順に返す。
func remainingExportIDs(t *testing.T, exportRepo *repository.ExportRepository, profileID model.ProfileID) []model.ExportID {
	t.Helper()

	exports, err := exportRepo.ListByProfileID(context.Background(), profileID, exportDeletionPageSizeForTests)
	if err != nil {
		t.Fatalf("プロフィールのエクスポートの取得に失敗: %v", err)
	}

	ids := make([]model.ExportID, len(exports))
	for i, export := range exports {
		ids[i] = export.ID
	}
	return ids
}

// assertExportIDs fails unless got holds exactly the wanted IDs in the given
// order.
//
// [Ja] assertExportIDs は got が指定順の want ID とちょうど一致しなければ失敗する。
func assertExportIDs(t *testing.T, label string, got []model.ExportID, want ...model.ExportID) {
	t.Helper()

	if !slices.Equal(got, want) {
		t.Errorf("%s = %v, want %v", label, got, want)
	}
}
