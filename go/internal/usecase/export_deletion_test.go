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

// exportDeletionPageSizeForTestsはこれらのテストが作るどのプロフィールより
// 大きく、検証が全件を見たい結果を一覧が切り詰めないようにする。
const exportDeletionPageSizeForTests int32 = 1000

// exportDeletionObjectStorageは、エクスポートの削除処理が呼ぶオブジェクト
// ストレージの代役。テストが置いたオブジェクトを保持し、何がどのキーから削除された
// かを記録する。指定したキーを拒否させられるため、一部が利用できないストレージで
// 生じる状態を観測できる。
type exportDeletionObjectStorage struct {
	t *testing.T

	// beforeDeleteは各削除の前に実行される。実行が候補と候補の間にいる間に、
	// テストがexportsテーブルを変更できるようにする。
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

// putはキーの位置にオブジェクトを保存する。エクスポートがアップロードした
// アーカイブを表す。
func (s *exportDeletionObjectStorage) put(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects[key] = struct{}{}
}

// failOnはストレージにそのキーを拒否させる。今は削除できないオブジェクトを
// 表す。
func (s *exportDeletionObjectStorage) failOn(key string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleteErrs[key] = err
}

// Deleteはオブジェクトを削除する。オブジェクトの無いキーには成功を返す。
// すでに存在しないオブジェクトの削除にS3アダプタが返すのと同じであり、これに
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
	// 削除処理は自分のエクスポートのオブジェクトを行そのものから知るため、一覧
	// 取得は、依頼されていないオブジェクトまで列挙することになる。
	s.t.Errorf("エクスポートの削除は ListPrefix を呼ばないはず (prefix: %s)", prefix)
	return nil
}

// deletedKeysは実行がオブジェクトを削除したキーを、削除した順に返す。
func (s *exportDeletionObjectStorage) deletedKeys() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.deleted)
}

// storedKeysはまだオブジェクトを保持しているキーを返す。
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

// buildStoredExportは対象に対して指定statusのエクスポートを作成し、その
// アーカイブを保存して、エクスポートIDとアーカイブのキーを返す。どのstatusでも
// 両方が存在するのは、試行がそれを記録する遷移より先にアップロードするためで、
// queued / started / failedのエクスポートもオブジェクトを持ちうる。
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

// remainingExportIDsはプロフィールのエクスポートのうち行が残っているものの
// IDを、古い順に返す。
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

// assertExportIDsはgotが指定順のwant IDとちょうど一致しなければ失敗する。
func assertExportIDs(t *testing.T, label string, got []model.ExportID, want ...model.ExportID) {
	t.Helper()

	if !slices.Equal(got, want) {
		t.Errorf("%s = %v、期待値 = %v", label, got, want)
	}
}
