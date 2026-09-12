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

// errExportObjectMissing stands for the storage answering that the object is
// gone, which is what a retried cleanup or an object removed outside the
// application looks like from here.
//
// [Ja] errExportObjectMissing は、オブジェクトが存在しないというストレージの応答を
// 表す。再試行中の cleanup や、アプリケーションの外で削除されたオブジェクトは、
// ここからはこの形で見える。
var errExportObjectMissing = errors.New("オブジェクトが存在しない")

// fakeExportDownloadStorage serves objects a test seeded and records the keys it
// was asked for. It is separate from fakeExportObjectStorage because that one
// exists to police the key an upload writes to, while a download reads an object
// it did not write and must not write anything at all: every other operation
// fails the test.
//
// [Ja] fakeExportDownloadStorage はテストが用意したオブジェクトを提供し、要求された
// キーを記録する。fakeExportObjectStorage と分けているのは、あちらがアップロードの
// 書き込み先のキーを取り締まるためのものであるのに対し、ダウンロードは自分が書いて
// いないオブジェクトを読むだけで、何も書いてはならないためである。ダウンロード以外の
// 操作はすべてテストの失敗にする。
type fakeExportDownloadStorage struct {
	t       *testing.T
	objects map[string][]byte

	// downloadErr replaces the object lookup when set, so a test can fail the
	// download itself instead of only leaving the object absent.
	//
	// [Ja] downloadErr を設定するとオブジェクトの検索を置き換える。テストは
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

// newSucceededExport creates a succeeded export whose object key follows
// the convention. The key embeds the export ID, which only exists after the
// insert, so it is written back rather than passed to the builder.
//
// [Ja] newSucceededExport は、オブジェクトキーが規約に従う succeeded な
// エクスポートを作る。キーはエクスポート ID を含むが、その ID は挿入後にしか
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

// readArchive reads the opened archive and closes it, so a test asserting on the
// bytes also leaves nothing open behind it.
//
// [Ja] readArchive は開いたアーカイブを読み切って閉じる。バイト列を検査するテストが、
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

// assertNotFound fails the test unless err refuses the request as not found,
// which is how both a profile the user does not own and a profile with nothing
// to download are answered.
//
// [Ja] assertNotFound は、err が not found としてリクエストを拒否していない場合に
// テストを失敗させる。所有していないプロフィールと、ダウンロードするものが無い
// プロフィールは、どちらもこの形で答えられる。
func assertNotFound(t *testing.T, err error) {
	t.Helper()

	appErr := model.AsAppError(err)
	if appErr == nil {
		t.Fatalf("Execute() error = %v, want *model.AppError", err)
	}
	if appErr.Code != model.AppErrCodeResourceNotFound {
		t.Errorf("Code = %v, want %v", appErr.Code, model.AppErrCodeResourceNotFound)
	}
}

// TestGetExportDownloadUsecase_Execute pins which archive a download hands over,
// who is allowed to ask for it, and how the refusals and failures differ.
//
// [Ja] TestGetExportDownloadUsecase_Execute は、ダウンロードがどのアーカイブを
// 渡すか、誰がそれを要求できるか、拒否と失敗がどう異なるかを固定する。
func TestGetExportDownloadUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	// 15:30 UTC is already the next day in Asia/Tokyo, the zone the fixture's
	// user is in, so the file name proves the date is read in the user's zone
	// rather than in UTC.
	//
	// [Ja] 15:30 UTC は、fixture のユーザーが属する Asia/Tokyo では既に翌日である。
	// そのためファイル名は、日付が UTC ではなくユーザーのゾーンで解釈されることを
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
			t.Fatalf("Execute() error = %v", err)
		}

		if got := readArchive(t, output.Body); !bytes.Equal(got, archive) {
			t.Errorf("Body = %q, want %q", got, archive)
		}
		if output.Size != int64(len(archive)) {
			t.Errorf("Size = %d, want %d", output.Size, len(archive))
		}
		if want := "mewst-export-20260724.zip"; output.FileName != want {
			t.Errorf("FileName = %q, want %q", output.FileName, want)
		}
		if want := []string{key}; !slices.Equal(storage.requestedKeys, want) {
			t.Errorf("requestedKeys = %v, want %v", storage.requestedKeys, want)
		}
	})

	// Cleanup deletes the archive a newer export replaced, and retries until it
	// succeeds. While that is outstanding the profile has two succeeded rows,
	// and only the newer one may be handed over.
	//
	// [Ja] cleanup は新しいエクスポートが置き換えたアーカイブを削除し、成功するまで
	// 再試行する。それが未完了の間、プロフィールは succeeded の行を 2 件持つが、
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
			t.Fatalf("Execute() error = %v", err)
		}

		if got := readArchive(t, output.Body); !bytes.Equal(got, latest) {
			t.Errorf("Body = %q, want %q", got, latest)
		}
		if want := []string{latestKey}; !slices.Equal(storage.requestedKeys, want) {
			t.Errorf("requestedKeys = %v, want %v", storage.requestedKeys, want)
		}
		if want := "mewst-export-20260725.zip"; output.FileName != want {
			t.Errorf("FileName = %q, want %q", output.FileName, want)
		}
	})

	t.Run("進行中のエクスポートしか無い場合は not found として拒否する", func(t *testing.T) {
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
			t.Errorf("Execute() output = %v, want nil", output)
		}
		assertNotFound(t, err)
		if len(storage.requestedKeys) != 0 {
			t.Errorf("requestedKeys = %v, want empty", storage.requestedKeys)
		}
	})

	t.Run("エクスポートが 1 件も無い場合は not found として拒否する", func(t *testing.T) {
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
			t.Errorf("Execute() output = %v, want nil", output)
		}
		assertNotFound(t, err)
		if len(storage.requestedKeys) != 0 {
			t.Errorf("requestedKeys = %v, want empty", storage.requestedKeys)
		}
	})

	t.Run("他のプロフィールのエクスポートは not found として拒否する", func(t *testing.T) {
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
			t.Errorf("Execute() output = %v, want nil", output)
		}
		assertNotFound(t, err)
		if len(storage.requestedKeys) != 0 {
			t.Errorf("requestedKeys = %v, want empty", storage.requestedKeys)
		}
	})

	// The object is missing while the row says it is there. Nothing about the
	// request is wrong, so this is reported as a failure rather than as a
	// refusal the reader could act on.
	//
	// [Ja] 行が存在すると言っているオブジェクトが欠けている状態。リクエストに誤りは
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
			t.Errorf("Execute() output = %v, want nil", output)
		}
		if !errors.Is(err, errExportObjectMissing) {
			t.Errorf("Execute() error = %v, want wrapped %v", err, errExportObjectMissing)
		}
		if appErr := model.AsAppError(err); appErr != nil {
			t.Errorf("Execute() error = %v, want no *model.AppError", appErr)
		}
		if want := []string{key}; !slices.Equal(storage.requestedKeys, want) {
			t.Errorf("requestedKeys = %v, want %v", storage.requestedKeys, want)
		}
	})

	// A reader who disconnects mid-download cancels the context, and the
	// cancellation must reach the caller as itself: reporting it as a missing
	// archive would tell the reader their export is gone.
	//
	// [Ja] ダウンロード中に切断した読み手は context をキャンセルする。その
	// キャンセルはそのままの形で呼び出し側へ届く必要がある。アーカイブが無いものと
	// して報告すると、読み手にはエクスポートが失われたと伝わってしまう。
	t.Run("context のキャンセルはそのままの失敗として返す", func(t *testing.T) {
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
			t.Errorf("Execute() output = %v, want nil", output)
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Execute() error = %v, want wrapped %v", err, context.Canceled)
		}
		if appErr := model.AsAppError(err); appErr != nil {
			t.Errorf("Execute() error = %v, want no *model.AppError", appErr)
		}
	})

	// A deployment without the object storage cannot serve an archive even for
	// an export an earlier deployment produced, so the request is closed before
	// the export is looked up at all.
	//
	// [Ja] オブジェクトストレージを持たないデプロイは、以前のデプロイが作った
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
			t.Errorf("Execute() output = %v, want nil", output)
		}

		appErr := model.AsAppError(err)
		if appErr == nil {
			t.Fatalf("Execute() error = %v, want *model.AppError", err)
		}
		if appErr.Code != model.AppErrCodeServiceUnavailable {
			t.Errorf("Code = %v, want %v", appErr.Code, model.AppErrCodeServiceUnavailable)
		}
		if len(storage.requestedKeys) != 0 {
			t.Errorf("requestedKeys = %v, want empty", storage.requestedKeys)
		}
	})

	// Authorization runs before the availability check, so a profile the user
	// does not own is refused the same way whether or not the deployment can
	// serve archives.
	//
	// [Ja] 認可は利用可否の判定より先に行う。ユーザーが所有していないプロフィールは、
	// そのデプロイがアーカイブを提供できるかどうかに関わらず同じ形で拒否される。
	t.Run("ストレージ未設定でも他のプロフィールは not found として拒否する", func(t *testing.T) {
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
			t.Errorf("Execute() output = %v, want nil", output)
		}
		assertNotFound(t, err)
	})
}
