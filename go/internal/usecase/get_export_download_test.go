package usecase_test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// errExportObjectMissingは、オブジェクトが存在しないというストレージの応答を
// 表す。再試行中のcleanupや、アプリケーションの外で削除されたオブジェクトは、
// ここからはこの形で見える。
var errExportObjectMissing = errors.New("オブジェクトが存在しない")

// fakeExportDownloadStorageはテストが用意したオブジェクトを提供し、要求された
// キーを記録する。fakeExportObjectStorageと分けているのは、あちらがアップロードの
// 書き込み先のキーを取り締まるためのものであるのに対し、ダウンロードは自分が書いて
// いないオブジェクトを読むだけで、何も書いてはならないためである。ダウンロード以外の
// 操作はすべてテストの失敗にする。
type fakeExportDownloadStorage struct {
	t       *testing.T
	objects map[string][]byte

	// downloadErrを設定するとオブジェクトの検索を置き換える。テストは
	// オブジェクトを欠けさせるだけでなく、ダウンロード自体を失敗させられる。
	downloadErr error

	requestedKeys []string
}

func newFakeExportDownloadStorage(t *testing.T) *fakeExportDownloadStorage {
	t.Helper()
	return &fakeExportDownloadStorage{t: t, objects: map[string][]byte{}}
}

func (f *fakeExportDownloadStorage) putObject(key string, data []byte) {
	f.objects[key] = data
}

func (f *fakeExportDownloadStorage) Download(_ context.Context, key string) (io.ReadCloser, int64, error) {
	f.requestedKeys = append(f.requestedKeys, key)

	if f.downloadErr != nil {
		return nil, 0, f.downloadErr
	}

	data, ok := f.objects[key]
	if !ok {
		return nil, 0, errExportObjectMissing
	}
	return io.NopCloser(bytes.NewReader(data)), int64(len(data)), nil
}

func (f *fakeExportDownloadStorage) Upload(_ context.Context, key string, _ io.Reader) error {
	f.t.Errorf("ダウンロードは Upload を呼ばないはず (key: %s)", key)
	return nil
}

func (f *fakeExportDownloadStorage) Delete(_ context.Context, key string) error {
	f.t.Errorf("ダウンロードは Delete を呼ばないはず (key: %s)", key)
	return nil
}

func (f *fakeExportDownloadStorage) ListPrefix(_ context.Context, prefix, _ string, _ func(key string, lastModified time.Time) error) error {
	f.t.Errorf("ダウンロードは ListPrefix を呼ばないはず (prefix: %s)", prefix)
	return nil
}

func newGetExportDownloadUsecase(
	t *testing.T,
	tx *sql.Tx,
	storage usecase.ExportObjectStorage,
	storageReady bool,
) *usecase.GetExportDownloadUsecase {
	t.Helper()

	queries := testutil.QueriesWithTx(tx)
	return usecase.NewGetExportDownloadUsecase(
		repository.NewUserProfileRepository(queries),
		repository.NewUserRepository(queries),
		repository.NewExportRepository(queries),
		storage,
		storageReady,
	)
}

// newSucceededExportは、オブジェクトキーが規約に従うsucceededな
// エクスポートを作る。キーはエクスポートIDを含むが、そのIDは挿入後にしか
// 存在しないため、ビルダーへ渡すのではなく後から書き戻す。
func newSucceededExport(t *testing.T, tx *sql.Tx, owner testutil.ProfileOwner, finishedAt time.Time) (model.ExportID, string) {
	t.Helper()

	id := testutil.NewExportBuilder(t, tx).
		WithProfileID(owner.ProfileID).
		WithActorID(owner.ActorID).
		WithStatus(model.ExportStatusSucceeded).
		WithCreatedAt(finishedAt).
		Build()

	key := usecase.ExportObjectKey(owner.ProfileID, id)
	if _, err := tx.Exec("UPDATE exports SET object_key = $1 WHERE id = $2", key, uuid.UUID(id)); err != nil {
		t.Fatalf("オブジェクトキーの更新に失敗: %v", err)
	}

	return id, key
}

// readArchiveは開いたアーカイブを読み切って閉じる。バイト列を検査するテストが、
// 開いたままのものを残さないようにするため。
func readArchive(t *testing.T, body io.ReadCloser) []byte {
	t.Helper()

	defer func() {
		if err := body.Close(); err != nil {
			t.Errorf("ストリームのクローズに失敗: %v", err)
		}
	}()

	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("ストリームの読み取りに失敗: %v", err)
	}
	return data
}

// assertNotFoundは、errがnot foundとしてリクエストを拒否していない場合に
// テストを失敗させる。所有していないプロフィールと、ダウンロードするものが無い
// プロフィールは、どちらもこの形で答えられる。
func assertNotFound(t *testing.T, err error) {
	t.Helper()

	appErr := model.AsAppError(err)
	if appErr == nil {
		t.Fatalf("Execute()のエラー = %v、*model.AppErrorを期待", err)
	}
	if appErr.Code != model.AppErrCodeResourceNotFound {
		t.Errorf("Code = %v、期待値 = %v", appErr.Code, model.AppErrCodeResourceNotFound)
	}
}

// TestGetExportDownloadUsecase_Executeは、ダウンロードがどのアーカイブを
// 渡すか、誰がそれを要求できるか、拒否と失敗がどう異なるかを固定する。
func TestGetExportDownloadUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	// 15:30 UTCは、fixtureのユーザーが属するAsia/Tokyoでは既に翌日である。
	// そのためファイル名は、日付がUTCではなくユーザーのゾーンで解釈されることを
	// 示す。
	finishedAt := time.Date(2026, 7, 23, 15, 30, 0, 0, time.UTC)

	t.Run("最新の成功したエクスポートのストリームとサイズとファイル名を返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		owner := testutil.NewProfileOwner(t, tx)
		_, key := newSucceededExport(t, tx, owner, finishedAt)

		storage := newFakeExportDownloadStorage(t)
		archive := []byte("PK\x03\x04 archive")
		storage.putObject(key, archive)

		uc := newGetExportDownloadUsecase(t, tx, storage, true)
		output, err := uc.Execute(ctx, usecase.GetExportDownloadInput{
			UserID:    owner.UserID,
			ProfileID: owner.ProfileID,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		if got := readArchive(t, output.Body); !bytes.Equal(got, archive) {
			t.Errorf("Body = %q、期待値 = %q", got, archive)
		}
		if output.Size != int64(len(archive)) {
			t.Errorf("Size = %d、期待値 = %d", output.Size, len(archive))
		}
		if want := "mewst-export-20260724.zip"; output.FileName != want {
			t.Errorf("FileName = %q、期待値 = %q", output.FileName, want)
		}
		if want := []string{key}; !slices.Equal(storage.requestedKeys, want) {
			t.Errorf("requestedKeys = %v、期待値 = %v", storage.requestedKeys, want)
		}
	})

	// cleanupは新しいエクスポートが置き換えたアーカイブを削除し、成功するまで
	// 再試行する。それが未完了の間、プロフィールはsucceededの行を2件持つが、
	// 渡してよいのは新しいほうだけである。
	t.Run("旧エクスポートの削除が再試行中でも最新の成功を渡す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		owner := testutil.NewProfileOwner(t, tx)
		_, oldKey := newSucceededExport(t, tx, owner, finishedAt)
		_, latestKey := newSucceededExport(t, tx, owner, finishedAt.Add(24*time.Hour))

		storage := newFakeExportDownloadStorage(t)
		storage.putObject(oldKey, []byte("old archive"))
		latest := []byte("latest archive")
		storage.putObject(latestKey, latest)

		uc := newGetExportDownloadUsecase(t, tx, storage, true)
		output, err := uc.Execute(ctx, usecase.GetExportDownloadInput{
			UserID:    owner.UserID,
			ProfileID: owner.ProfileID,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		if got := readArchive(t, output.Body); !bytes.Equal(got, latest) {
			t.Errorf("Body = %q、期待値 = %q", got, latest)
		}
		if want := []string{latestKey}; !slices.Equal(storage.requestedKeys, want) {
			t.Errorf("requestedKeys = %v、期待値 = %v", storage.requestedKeys, want)
		}
		if want := "mewst-export-20260725.zip"; output.FileName != want {
			t.Errorf("FileName = %q、期待値 = %q", output.FileName, want)
		}
	})

	t.Run("進行中のエクスポートしか無い場合はnot foundとして拒否する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		owner := testutil.NewProfileOwner(t, tx)
		testutil.NewExportBuilder(t, tx).
			WithProfileID(owner.ProfileID).
			WithActorID(owner.ActorID).
			WithStatus(model.ExportStatusStarted).
			WithCreatedAt(finishedAt).
			Build()

		storage := newFakeExportDownloadStorage(t)
		uc := newGetExportDownloadUsecase(t, tx, storage, true)

		output, err := uc.Execute(ctx, usecase.GetExportDownloadInput{
			UserID:    owner.UserID,
			ProfileID: owner.ProfileID,
		})
		if output != nil {
			t.Errorf("Execute()の出力 = %v、期待値 = nil", output)
		}
		assertNotFound(t, err)
		if len(storage.requestedKeys) != 0 {
			t.Errorf("requestedKeys = %v、空を期待", storage.requestedKeys)
		}
	})

	t.Run("エクスポートが1件も無い場合はnot foundとして拒否する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		owner := testutil.NewProfileOwner(t, tx)

		storage := newFakeExportDownloadStorage(t)
		uc := newGetExportDownloadUsecase(t, tx, storage, true)

		output, err := uc.Execute(ctx, usecase.GetExportDownloadInput{
			UserID:    owner.UserID,
			ProfileID: owner.ProfileID,
		})
		if output != nil {
			t.Errorf("Execute()の出力 = %v、期待値 = nil", output)
		}
		assertNotFound(t, err)
		if len(storage.requestedKeys) != 0 {
			t.Errorf("requestedKeys = %v、空を期待", storage.requestedKeys)
		}
	})

	t.Run("他のプロフィールのエクスポートはnot foundとして拒否する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		owner := testutil.NewProfileOwner(t, tx)
		other := testutil.NewProfileOwner(t, tx)
		_, key := newSucceededExport(t, tx, owner, finishedAt)

		storage := newFakeExportDownloadStorage(t)
		storage.putObject(key, []byte("archive"))

		uc := newGetExportDownloadUsecase(t, tx, storage, true)
		output, err := uc.Execute(ctx, usecase.GetExportDownloadInput{
			UserID:    other.UserID,
			ProfileID: owner.ProfileID,
		})
		if output != nil {
			t.Errorf("Execute()の出力 = %v、期待値 = nil", output)
		}
		assertNotFound(t, err)
		if len(storage.requestedKeys) != 0 {
			t.Errorf("requestedKeys = %v、空を期待", storage.requestedKeys)
		}
	})

	// 行が存在すると言っているオブジェクトが欠けている状態。リクエストに誤りは
	// 無いため、読み手が対処できる拒否ではなく失敗として報告する。
	t.Run("オブジェクトが存在しない場合は拒否ではなく失敗として返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		owner := testutil.NewProfileOwner(t, tx)
		_, key := newSucceededExport(t, tx, owner, finishedAt)

		storage := newFakeExportDownloadStorage(t)
		uc := newGetExportDownloadUsecase(t, tx, storage, true)

		output, err := uc.Execute(ctx, usecase.GetExportDownloadInput{
			UserID:    owner.UserID,
			ProfileID: owner.ProfileID,
		})
		if output != nil {
			t.Errorf("Execute()の出力 = %v、期待値 = nil", output)
		}
		if !errors.Is(err, errExportObjectMissing) {
			t.Errorf("Execute()のエラー = %v、%vをラップしたエラーを期待", err, errExportObjectMissing)
		}
		if appErr := model.AsAppError(err); appErr != nil {
			t.Errorf("Execute()のエラー = %v、*model.AppErrorではないエラーを期待", appErr)
		}
		if want := []string{key}; !slices.Equal(storage.requestedKeys, want) {
			t.Errorf("requestedKeys = %v、期待値 = %v", storage.requestedKeys, want)
		}
	})

	// ダウンロード中に切断した読み手はcontextをキャンセルする。その
	// キャンセルはそのままの形で呼び出し側へ届く必要がある。アーカイブが無いものと
	// して報告すると、読み手にはエクスポートが失われたと伝わってしまう。
	t.Run("contextのキャンセルはそのままの失敗として返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		owner := testutil.NewProfileOwner(t, tx)
		_, key := newSucceededExport(t, tx, owner, finishedAt)

		storage := newFakeExportDownloadStorage(t)
		storage.putObject(key, []byte("archive"))
		storage.downloadErr = context.Canceled

		uc := newGetExportDownloadUsecase(t, tx, storage, true)
		output, err := uc.Execute(ctx, usecase.GetExportDownloadInput{
			UserID:    owner.UserID,
			ProfileID: owner.ProfileID,
		})
		if output != nil {
			t.Errorf("Execute()の出力 = %v、期待値 = nil", output)
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Execute()のエラー = %v、%vをラップしたエラーを期待", err, context.Canceled)
		}
		if appErr := model.AsAppError(err); appErr != nil {
			t.Errorf("Execute()のエラー = %v、*model.AppErrorではないエラーを期待", appErr)
		}
	})

	// オブジェクトストレージを持たないデプロイは、以前のデプロイが作った
	// エクスポートであってもアーカイブを提供できない。そのため、エクスポートを
	// 参照する前にリクエストを閉じる。
	t.Run("ストレージ未設定では利用不可として拒否し、ストレージへ触れない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		owner := testutil.NewProfileOwner(t, tx)
		_, key := newSucceededExport(t, tx, owner, finishedAt)

		storage := newFakeExportDownloadStorage(t)
		storage.putObject(key, []byte("archive"))

		uc := newGetExportDownloadUsecase(t, tx, storage, false)
		output, err := uc.Execute(ctx, usecase.GetExportDownloadInput{
			UserID:    owner.UserID,
			ProfileID: owner.ProfileID,
		})
		if output != nil {
			t.Errorf("Execute()の出力 = %v、期待値 = nil", output)
		}

		appErr := model.AsAppError(err)
		if appErr == nil {
			t.Fatalf("Execute()のエラー = %v、*model.AppErrorを期待", err)
		}
		if appErr.Code != model.AppErrCodeServiceUnavailable {
			t.Errorf("Code = %v、期待値 = %v", appErr.Code, model.AppErrCodeServiceUnavailable)
		}
		if len(storage.requestedKeys) != 0 {
			t.Errorf("requestedKeys = %v、空を期待", storage.requestedKeys)
		}
	})

	// 認可は利用可否の判定より先に行う。ユーザーが所有していないプロフィールは、
	// そのデプロイがアーカイブを提供できるかどうかに関わらず同じ形で拒否される。
	t.Run("ストレージ未設定でも他のプロフィールはnot foundとして拒否する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		owner := testutil.NewProfileOwner(t, tx)
		other := testutil.NewProfileOwner(t, tx)
		newSucceededExport(t, tx, owner, finishedAt)

		storage := newFakeExportDownloadStorage(t)
		uc := newGetExportDownloadUsecase(t, tx, storage, false)

		output, err := uc.Execute(ctx, usecase.GetExportDownloadInput{
			UserID:    other.UserID,
			ProfileID: owner.ProfileID,
		})
		if output != nil {
			t.Errorf("Execute()の出力 = %v、期待値 = nil", output)
		}
		assertNotFound(t, err)
	})
}
