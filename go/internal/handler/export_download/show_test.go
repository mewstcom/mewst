package export_download_test

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/mewstcom/mewst/go/internal/handler/export_download"
	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// finishedAtはfixtureのエクスポートが完了した時刻。15:30 UTCはfixtureの
// ユーザーが属するAsia/Tokyoでは既に翌日であるため、レスポンスが提示するファイル名は
// 日付がユーザーのゾーンから来ていることを示す。
var finishedAt = time.Date(2026, 7, 23, 15, 30, 0, 0, time.UTC)

// wantFileNameはfixtureのユーザーのゾーンでfinishedAtが導くファイル名。
const wantFileName = "mewst-export-20260724.zip"

// trackedArchiveは閉じられたかどうかを記録するアーカイブのストリーム。
// ハンドラーは渡されたストリームを必ず閉じなければならず、読み手が途中で中断した
// ものも例外ではない。各テストがそれを検査できるようにし、開いたままのR2接続を
// 本番で気付く形にしない。
type trackedArchive struct {
	reader io.Reader
	closed chan struct{}
	once   sync.Once
}

func newTrackedArchive(reader io.Reader) *trackedArchive {
	return &trackedArchive{reader: reader, closed: make(chan struct{})}
}

func (a *trackedArchive) Read(p []byte) (int, error) {
	return a.reader.Read(p)
}

func (a *trackedArchive) Close() error {
	a.once.Do(func() { close(a.closed) })
	return nil
}

func (a *trackedArchive) isClosed() bool {
	select {
	case <-a.closed:
		return true
	default:
		return false
	}
}

// awaitClosedはハンドラーがストリームを閉じるのを待つ。実サーバーを動かす
// テストが、ハンドラーが気付くまでの時間を当てずに済むようにする。
func (a *trackedArchive) awaitClosed(t *testing.T) {
	t.Helper()

	select {
	case <-a.closed:
	case <-time.After(5 * time.Second):
		t.Fatal("ハンドラーがストリームを閉じませんでした")
	}
}

// haltingReaderは保持しているバイト列を渡した後、ダウンロードのcontextが
// 終わるのを待つ。読み手が去ったときのオブジェクトのストリームはこう振る舞う
// (リクエストのcontextがキャンセルされ、R2のボディの次の読み取りが失敗する)。
type haltingReader struct {
	ctx    context.Context
	prefix *bytes.Reader
}

func (r *haltingReader) Read(p []byte) (int, error) {
	if r.prefix.Len() > 0 {
		return r.prefix.Read(p)
	}

	<-r.ctx.Done()
	return 0, r.ctx.Err()
}

// fakeExportStorageはテストが用意した1件のアーカイブを提供し、要求された
// キーを記録する。動作する操作はDownloadだけである。アーカイブを渡す処理は
// オブジェクトを読むだけで何も書かないため、それ以外の操作はテストの失敗にする。
type fakeExportStorage struct {
	t    *testing.T
	key  string
	size int64

	// newReaderはダウンロードのボディを組み立てる。ダウンロードのcontextを
	// 受け取るため、読み手が去ったときに終わるストリームをテストが再現できる。
	newReader func(ctx context.Context) io.Reader

	// muは、テストが自身のgoroutineから観測している間にハンドラーが書き込む
	// フィールドを保護する。
	mu            sync.Mutex
	archive       *trackedArchive
	requestedKeys []string
}

func newFakeExportStorage(t *testing.T, key string, archive []byte) *fakeExportStorage {
	t.Helper()

	return &fakeExportStorage{
		t:         t,
		key:       key,
		size:      int64(len(archive)),
		newReader: func(context.Context) io.Reader { return bytes.NewReader(archive) },
	}
}

func (f *fakeExportStorage) Download(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.requestedKeys = append(f.requestedKeys, key)
	if key != f.key {
		return nil, 0, fmt.Errorf("オブジェクトが存在しない (key: %s)", key)
	}

	f.archive = newTrackedArchive(f.newReader(ctx))
	return f.archive, f.size, nil
}

func (f *fakeExportStorage) Upload(_ context.Context, key string, _ io.Reader) error {
	f.t.Errorf("ダウンロードは Upload を呼ばないはず (key: %s)", key)
	return nil
}

func (f *fakeExportStorage) Delete(_ context.Context, key string) error {
	f.t.Errorf("ダウンロードは Delete を呼ばないはず (key: %s)", key)
	return nil
}

func (f *fakeExportStorage) ListPrefix(_ context.Context, prefix, _ string, _ func(key string, lastModified time.Time) error) error {
	f.t.Errorf("ダウンロードは ListPrefix を呼ばないはず (prefix: %s)", prefix)
	return nil
}

// handedOverはストレージが開いたストリームを返す。開いていない場合はnil。
func (f *fakeExportStorage) handedOver() *trackedArchive {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.archive
}

func (f *fakeExportStorage) keys() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.requestedKeys...)
}

// newHandlerはUseCaseがテストのtransaction内で動くHandlerを構築する。
func newHandler(t *testing.T, tx *sql.Tx, storage usecase.ExportObjectStorage, storageReady bool) *export_download.Handler {
	t.Helper()

	queries := testutil.QueriesWithTx(tx)
	return export_download.NewHandler(usecase.NewGetExportDownloadUsecase(
		repository.NewUserProfileRepository(queries),
		repository.NewUserRepository(queries),
		repository.NewExportRepository(queries),
		storage,
		storageReady,
	))
}

// newSucceededExportはプロフィールのダウンロード可能なエクスポートを作り、
// そのアーカイブが保存されているキーを返す。
func newSucceededExport(t *testing.T, tx *sql.Tx, owner testutil.ProfileOwner) string {
	t.Helper()

	key := fmt.Sprintf("exports/%s/archive.zip", owner.ProfileID)
	testutil.NewExportBuilder(t, tx).
		WithProfileID(owner.ProfileID).
		WithActorID(owner.ActorID).
		WithStatus(model.ExportStatusSucceeded).
		WithObjectKey(key).
		WithCreatedAt(finishedAt).
		Build()

	return key
}

// signedInContextは本番でi18n / RequireAuthミドルウェアが渡すもの
// (ロケールと、ログイン中のユーザーとプロフィール) をparentへ足す。新しいcontext
// ではなくリクエスト自身のcontextを土台にするのは、読み手が去ったときにサーバーが
// 通知するキャンセルを運ぶのがそれであるため。
func signedInContext(parent context.Context, owner testutil.ProfileOwner) context.Context {
	ctx := i18n.SetLocale(parent, "ja")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: owner.UserID})
	return middleware.SetProfileToContext(ctx, &model.Profile{ID: owner.ProfileID, Atname: "alice"})
}

func newDownloadRequest(owner testutil.ProfileOwner) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/settings/export/download", nil)
	return req.WithContext(signedInContext(req.Context(), owner))
}

// newArchiveは実際のzipを組み立てる。ファイルアプリが本当に開けるバイト列に
// 対してテストが検査するようにし、どんな変換を経ても読めてしまうプレースホルダーを
// 使わないため。
func newArchive(t *testing.T) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("index.html")
	if err != nil {
		t.Fatalf("zipエントリの作成に失敗: %v", err)
	}
	if _, err := w.Write([]byte("<!doctype html><title>エクスポート</title>")); err != nil {
		t.Fatalf("zipエントリの書き込みに失敗: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zipのクローズに失敗: %v", err)
	}

	return buf.Bytes()
}

// assertReadableArchiveは、bodyがfixtureの書いたエントリを持つzipのままで
// なければテストを失敗させる。転送が再エンコードしていないことを示すため。
func assertReadableArchive(t *testing.T, body []byte) {
	t.Helper()

	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatalf("レスポンスをzipとして読めません: %v", err)
	}
	if len(zr.File) != 1 || zr.File[0].Name != "index.html" {
		t.Errorf("zipの内容が不正: %v", zr.File)
	}
}

// assertArchiveHeadersはレスポンスがアーカイブに与える説明を固定する。
// メディアタイプ、ファイルアプリが表示する名前、sniffingの拒否、保存の禁止である。
func assertArchiveHeaders(t *testing.T, header http.Header, size int) {
	t.Helper()

	wants := map[string]string{
		"Content-Type":           "application/zip",
		"Content-Disposition":    fmt.Sprintf("attachment; filename=%q", wantFileName),
		"X-Content-Type-Options": "nosniff",
		"Cache-Control":          "private, no-store",
		"Content-Length":         strconv.Itoa(size),
	}
	for name, want := range wants {
		if got := header.Get(name); got != want {
			t.Errorf("%s = %q、期待値 = %q", name, got, want)
		}
	}
	// アーカイブは既に圧縮されているため、経路上のどこもこれを再エンコード
	// してはならない。
	if got := header.Get("Content-Encoding"); got != "" {
		t.Errorf("Content-Encoding = %q、空を期待", got)
	}
}

// TestShow_HandsOverArchiveは、ダウンロード可能なエクスポートを持つ読み手が
// 受け取るレスポンスを固定する。アーカイブそのものが、ファイルアプリがエクスポートの
// 名前で保存し、どこも複製を保持しない形で説明されて返る。
func TestShow_HandsOverArchive(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	owner := testutil.NewProfileOwner(t, tx)
	key := newSucceededExport(t, tx, owner)

	archive := newArchive(t)
	storage := newFakeExportStorage(t, key, archive)
	h := newHandler(t, tx, storage, true)

	rr := httptest.NewRecorder()
	h.Show(rr, newDownloadRequest(owner))

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusOK)
	}
	assertArchiveHeaders(t, rr.Header(), len(archive))

	if got := rr.Body.Bytes(); !bytes.Equal(got, archive) {
		t.Errorf("レスポンスのバイト列がアーカイブと一致しません (実測値 = %dバイト、期待値 = %dバイト)", len(got), len(archive))
	}
	assertReadableArchive(t, rr.Body.Bytes())

	if got := storage.keys(); len(got) != 1 || got[0] != key {
		t.Errorf("requestedKeys = %v、期待値 = %v", got, []string{key})
	}
	if handedOver := storage.handedOver(); handedOver == nil || !handedOver.isClosed() {
		t.Error("ハンドラーがストリームを閉じていません")
	}
}

// TestShow_ClosesStreamWhenClientDisconnectsは実サーバーを動かし、切断を
// テストではなくトランスポートのものにする。ダウンロードを途中で中断した読み手が、
// 開いたままのオブジェクトのストリームを残さないことを確かめる。
func TestShow_ClosesStreamWhenClientDisconnects(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	owner := testutil.NewProfileOwner(t, tx)
	key := newSucceededExport(t, tx, owner)

	// prefixはサーバーの書き込みバッファを越えてクライアントへ届く大きさにし、
	// 宣言するサイズはそれより大きくする。これにより読み手は、残りがある状態で
	// 切断される。
	prefix := bytes.Repeat([]byte("PK"), 32*1024)
	storage := newFakeExportStorage(t, key, prefix)
	storage.size = int64(len(prefix)) + 1024
	storage.newReader = func(ctx context.Context) io.Reader {
		return &haltingReader{ctx: ctx, prefix: bytes.NewReader(prefix)}
	}

	srv := httptest.NewServer(newServerChain(t, tx, storage, owner))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/settings/export/download", nil)
	if err != nil {
		t.Fatalf("リクエストの作成に失敗: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("リクエストの送信に失敗: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if _, err := io.ReadFull(resp.Body, make([]byte, 1024)); err != nil {
		t.Fatalf("レスポンスの先頭の読み取りに失敗: %v", err)
	}

	cancel()

	handedOver := storage.handedOver()
	if handedOver == nil {
		t.Fatal("ストレージからストリームを取得していません")
	}
	handedOver.awaitClosed(t)
}

// TestShow_WithoutDownloadableExportは、存在しないアーカイブへのリクエストに、
// 所有していないプロフィールと同じnot foundで答え、ストレージには触れない。
func TestShow_WithoutDownloadableExport(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	owner := testutil.NewProfileOwner(t, tx)

	storage := newFakeExportStorage(t, "unused", nil)
	h := newHandler(t, tx, storage, true)

	rr := httptest.NewRecorder()
	h.Show(rr, newDownloadRequest(owner))

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
	if got := rr.Header().Get("Content-Disposition"); got != "" {
		t.Errorf("Content-Disposition = %q、空を期待", got)
	}
	if got := storage.keys(); len(got) != 0 {
		t.Errorf("requestedKeys = %v、空を期待", got)
	}
}

// TestShow_OtherProfilesExportは、所有していないプロフィールを要求した読み手を
// 拒否する。アーカイブが存在することを応答から読み取れないようにする。
func TestShow_OtherProfilesExport(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	target := testutil.NewProfileOwner(t, tx)
	key := newSucceededExport(t, tx, target)
	other := testutil.NewProfileOwner(t, tx)

	storage := newFakeExportStorage(t, key, newArchive(t))
	h := newHandler(t, tx, storage, true)

	// ログイン中の読み手は別のユーザーだが、リクエストは対象のプロフィールを
	// 指す。組のどちらか一方だけでアーカイブを開けてはならない。
	req := newDownloadRequest(testutil.ProfileOwner{UserID: other.UserID, ProfileID: target.ProfileID})

	rr := httptest.NewRecorder()
	h.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
	if got := storage.keys(); len(got) != 0 {
		t.Errorf("requestedKeys = %v、空を期待", got)
	}
}

// TestShow_WhenExportsUnavailableは、オブジェクトストレージを持たないデプロイで
// リクエストを閉じる。提供できない機能を説明する代わりに失敗させる。
func TestShow_WhenExportsUnavailable(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	owner := testutil.NewProfileOwner(t, tx)
	newSucceededExport(t, tx, owner)

	h := newHandler(t, tx, nil, false)

	rr := httptest.NewRecorder()
	h.Show(rr, newDownloadRequest(owner))

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusServiceUnavailable)
	}
	if got := rr.Header().Get("Content-Disposition"); got != "" {
		t.Errorf("Content-Disposition = %q、空を期待", got)
	}
}

// TestShow_WithoutSignedInIdentityは、RequireAuth無しでルートが登録された
// 場合にリクエストを失敗させる。所有者が不明なままアーカイブを開かないため。
func TestShow_WithoutSignedInIdentity(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	owner := testutil.NewProfileOwner(t, tx)
	key := newSucceededExport(t, tx, owner)

	storage := newFakeExportStorage(t, key, newArchive(t))
	h := newHandler(t, tx, storage, true)

	rr := httptest.NewRecorder()
	h.Show(rr, httptest.NewRequest(http.MethodGet, "/settings/export/download", nil))

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusInternalServerError)
	}
	if got := storage.keys(); len(got) != 0 {
		t.Errorf("requestedKeys = %v、空を期待", got)
	}
}

// newServerChainはダウンロードのルートが動くミドルウェアでハンドラーを包み、
// RequireAuthが解決したはずのログイン中のidentityを載せる。PrivateCacheを
// 含めるのは、ダウンロードされるファイルのためにハンドラーが上書きしなければならない
// Cache-Controlを設定するミドルウェアであるため。
func newServerChain(t *testing.T, tx *sql.Tx, storage usecase.ExportObjectStorage, owner testutil.ProfileOwner) http.Handler {
	t.Helper()

	h := newHandler(t, tx, storage, true)
	return middleware.PrivateCache(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.Show(w, r.WithContext(signedInContext(r.Context(), owner)))
	}))
}

// TestShow_ThroughReverseProxyはダウンロードをリバースプロキシ越しに送る。
// 開発環境でも本番環境でもアプリの前段にはプロキシが置かれるため、これがリクエストの
// 実際の形である。読み手が最終的に得るものは、そのホップを経ても同じアーカイブで、
// 同じヘッダーで説明されていなければならない。
func TestShow_ThroughReverseProxy(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	owner := testutil.NewProfileOwner(t, tx)
	key := newSucceededExport(t, tx, owner)

	archive := newArchive(t)
	storage := newFakeExportStorage(t, key, archive)

	origin := httptest.NewServer(newServerChain(t, tx, storage, owner))
	defer origin.Close()

	originURL, err := url.Parse(origin.URL)
	if err != nil {
		t.Fatalf("オリジンのURLの解析に失敗: %v", err)
	}
	proxy := httptest.NewServer(httputil.NewSingleHostReverseProxy(originURL))
	defer proxy.Close()

	resp, err := http.Get(proxy.URL + "/settings/export/download")
	if err != nil {
		t.Fatalf("リクエストの送信に失敗: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ステータスコードが不正: 実測値 = %v、期待値 = %v", resp.StatusCode, http.StatusOK)
	}
	assertArchiveHeaders(t, resp.Header, len(archive))

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("レスポンスの読み取りに失敗: %v", err)
	}
	if !bytes.Equal(body, archive) {
		t.Errorf("レスポンスのバイト列がアーカイブと一致しません (実測値 = %dバイト、期待値 = %dバイト)", len(body), len(archive))
	}
	assertReadableArchive(t, body)

	if handedOver := storage.handedOver(); handedOver == nil || !handedOver.isClosed() {
		t.Error("ハンドラーがストリームを閉じていません")
	}
}
