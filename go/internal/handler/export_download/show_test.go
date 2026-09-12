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

// finishedAt is when the fixture's export completed. 15:30 UTC is already the
// next day in Asia/Tokyo, the zone the fixture's user is in, so the file name
// the response offers proves the date came from the user's zone.
//
// [Ja] finishedAt は fixture のエクスポートが完了した時刻。15:30 UTC は fixture の
// ユーザーが属する Asia/Tokyo では既に翌日であるため、レスポンスが提示するファイル名は
// 日付がユーザーのゾーンから来ていることを示す。
var finishedAt = time.Date(2026, 7, 23, 15, 30, 0, 0, time.UTC)

// wantFileName is the name finishedAt yields in the fixture user's zone.
//
// [Ja] wantFileName は fixture のユーザーのゾーンで finishedAt が導くファイル名。
const wantFileName = "mewst-export-20260724.zip"

// trackedArchive is an archive stream that records whether it was closed. The
// handler must close every stream it is handed, including the one a reader
// abandons halfway, so each test can assert on that rather than leaving an open
// R2 connection to be noticed in production.
//
// [Ja] trackedArchive は閉じられたかどうかを記録するアーカイブのストリーム。
// ハンドラーは渡されたストリームを必ず閉じなければならず、読み手が途中で中断した
// ものも例外ではない。各テストがそれを検査できるようにし、開いたままの R2 接続を
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

// awaitClosed waits for the handler to close the stream, so a test driving a
// real server does not have to guess how long the handler needs to notice.
//
// [Ja] awaitClosed はハンドラーがストリームを閉じるのを待つ。実サーバーを動かす
// テストが、ハンドラーが気付くまでの時間を当てずに済むようにする。
func (a *trackedArchive) awaitClosed(t *testing.T) {
	t.Helper()

	select {
	case <-a.closed:
	case <-time.After(5 * time.Second):
		t.Fatal("ハンドラーがストリームを閉じませんでした")
	}
}

// haltingReader hands over its buffered bytes and then waits for the download's
// context to end, which is what an object stream does when the reader goes away:
// the request context is cancelled and the next read of the R2 body fails.
//
// [Ja] haltingReader は保持しているバイト列を渡した後、ダウンロードの context が
// 終わるのを待つ。読み手が去ったときのオブジェクトのストリームはこう振る舞う
// (リクエストの context がキャンセルされ、R2 のボディの次の読み取りが失敗する)。
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

// fakeExportStorage serves the one archive a test seeded and records the keys it
// was asked for. Only Download is implemented as a working operation: handing
// over an archive reads an object and writes nothing, so every other operation
// fails the test.
//
// [Ja] fakeExportStorage はテストが用意した 1 件のアーカイブを提供し、要求された
// キーを記録する。動作する操作は Download だけである。アーカイブを渡す処理は
// オブジェクトを読むだけで何も書かないため、それ以外の操作はテストの失敗にする。
type fakeExportStorage struct {
	t    *testing.T
	key  string
	size int64

	// newReader builds the body for a download. It receives the download's
	// context so a test can model a stream that ends when the reader leaves.
	//
	// [Ja] newReader はダウンロードのボディを組み立てる。ダウンロードの context を
	// 受け取るため、読み手が去ったときに終わるストリームをテストが再現できる。
	newReader func(ctx context.Context) io.Reader

	// mu guards the fields the handler writes while a test observes them from
	// its own goroutine.
	//
	// [Ja] mu は、テストが自身の goroutine から観測している間にハンドラーが書き込む
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

// handedOver returns the stream the storage opened, or nil if it never opened
// one.
//
// [Ja] handedOver はストレージが開いたストリームを返す。開いていない場合は nil。
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

// newHandler builds a Handler whose UseCase runs inside the test's transaction.
//
// [Ja] newHandler は UseCase がテストの transaction 内で動く Handler を構築する。
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

// newSucceededExport creates the profile's downloadable export and returns the
// key its archive is stored under.
//
// [Ja] newSucceededExport はプロフィールのダウンロード可能なエクスポートを作り、
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

// signedInContext adds to parent what the i18n and RequireAuth middleware
// supply in production: the locale and the signed-in user and profile. It
// builds on the request's own context rather than on a fresh one, because that
// is what carries the cancellation the server signals when a reader leaves.
//
// [Ja] signedInContext は本番で i18n / RequireAuth ミドルウェアが渡すもの
// (ロケールと、ログイン中のユーザーとプロフィール) を parent へ足す。新しい context
// ではなくリクエスト自身の context を土台にするのは、読み手が去ったときにサーバーが
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

// newArchive builds a real zip so the tests assert on bytes a file manager can
// actually open, rather than on a placeholder that any transformation would
// still leave readable.
//
// [Ja] newArchive は実際の zip を組み立てる。ファイルアプリが本当に開けるバイト列に
// 対してテストが検査するようにし、どんな変換を経ても読めてしまうプレースホルダーを
// 使わないため。
func newArchive(t *testing.T) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("index.html")
	if err != nil {
		t.Fatalf("zip エントリの作成に失敗: %v", err)
	}
	if _, err := w.Write([]byte("<!doctype html><title>エクスポート</title>")); err != nil {
		t.Fatalf("zip エントリの書き込みに失敗: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip のクローズに失敗: %v", err)
	}

	return buf.Bytes()
}

// assertReadableArchive fails unless body is still a zip holding the entry the
// fixture wrote, which is what proves the transfer did not re-encode it.
//
// [Ja] assertReadableArchive は、body が fixture の書いたエントリを持つ zip のままで
// なければテストを失敗させる。転送が再エンコードしていないことを示すため。
func assertReadableArchive(t *testing.T, body []byte) {
	t.Helper()

	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatalf("レスポンスを zip として読めません: %v", err)
	}
	if len(zr.File) != 1 || zr.File[0].Name != "index.html" {
		t.Errorf("zip の内容が不正: %v", zr.File)
	}
}

// assertArchiveHeaders pins the description the response gives the archive: its
// media type, the name a file manager shows, the refusal to sniff, and the ban
// on storing it.
//
// [Ja] assertArchiveHeaders はレスポンスがアーカイブに与える説明を固定する。
// メディアタイプ、ファイルアプリが表示する名前、sniffing の拒否、保存の禁止である。
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
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	// The archive is already compressed, so nothing on the path may re-encode
	// it on the way out.
	//
	// [Ja] アーカイブは既に圧縮されているため、経路上のどこもこれを再エンコード
	// してはならない。
	if got := header.Get("Content-Encoding"); got != "" {
		t.Errorf("Content-Encoding = %q, want empty", got)
	}
}

// TestShow_HandsOverArchive pins the response a reader with a downloadable
// export receives: the archive itself, described so that a file manager saves
// it under the export's name and nothing keeps a copy.
//
// [Ja] TestShow_HandsOverArchive は、ダウンロード可能なエクスポートを持つ読み手が
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
		t.Fatalf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusOK)
	}
	assertArchiveHeaders(t, rr.Header(), len(archive))

	if got := rr.Body.Bytes(); !bytes.Equal(got, archive) {
		t.Errorf("レスポンスのバイト列がアーカイブと一致しません (got %d bytes, want %d bytes)", len(got), len(archive))
	}
	assertReadableArchive(t, rr.Body.Bytes())

	if got := storage.keys(); len(got) != 1 || got[0] != key {
		t.Errorf("requestedKeys = %v, want %v", got, []string{key})
	}
	if handedOver := storage.handedOver(); handedOver == nil || !handedOver.isClosed() {
		t.Error("ハンドラーがストリームを閉じていません")
	}
}

// TestShow_ClosesStreamWhenClientDisconnects drives a real server so the
// disconnect is the transport's, not the test's: a reader who cancels a download
// halfway must leave no open object stream behind.
//
// [Ja] TestShow_ClosesStreamWhenClientDisconnects は実サーバーを動かし、切断を
// テストではなくトランスポートのものにする。ダウンロードを途中で中断した読み手が、
// 開いたままのオブジェクトのストリームを残さないことを確かめる。
func TestShow_ClosesStreamWhenClientDisconnects(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	owner := testutil.NewProfileOwner(t, tx)
	key := newSucceededExport(t, tx, owner)

	// The prefix is large enough to reach the client through the server's write
	// buffer, and the declared size is larger still, so the reader is cut off
	// with bytes outstanding.
	//
	// [Ja] prefix はサーバーの書き込みバッファを越えてクライアントへ届く大きさにし、
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

// TestShow_WithoutDownloadableExport answers a request for an archive that does
// not exist with the same not found an unowned profile receives, and leaves the
// storage untouched.
//
// [Ja] TestShow_WithoutDownloadableExport は、存在しないアーカイブへのリクエストに、
// 所有していないプロフィールと同じ not found で答え、ストレージには触れない。
func TestShow_WithoutDownloadableExport(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	owner := testutil.NewProfileOwner(t, tx)

	storage := newFakeExportStorage(t, "unused", nil)
	h := newHandler(t, tx, storage, true)

	rr := httptest.NewRecorder()
	h.Show(rr, newDownloadRequest(owner))

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusNotFound)
	}
	if got := rr.Header().Get("Content-Disposition"); got != "" {
		t.Errorf("Content-Disposition = %q, want empty", got)
	}
	if got := storage.keys(); len(got) != 0 {
		t.Errorf("requestedKeys = %v, want empty", got)
	}
}

// TestShow_OtherProfilesExport refuses a reader asking for a profile they do not
// own, without letting the response reveal that the archive exists.
//
// [Ja] TestShow_OtherProfilesExport は、所有していないプロフィールを要求した読み手を
// 拒否する。アーカイブが存在することを応答から読み取れないようにする。
func TestShow_OtherProfilesExport(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	target := testutil.NewProfileOwner(t, tx)
	key := newSucceededExport(t, tx, target)
	other := testutil.NewProfileOwner(t, tx)

	storage := newFakeExportStorage(t, key, newArchive(t))
	h := newHandler(t, tx, storage, true)

	// The signed-in reader is the other user, but the request names the target's
	// profile: neither half of the pair alone may open the archive.
	//
	// [Ja] ログイン中の読み手は別のユーザーだが、リクエストは対象のプロフィールを
	// 指す。組のどちらか一方だけでアーカイブを開けてはならない。
	req := newDownloadRequest(testutil.ProfileOwner{UserID: other.UserID, ProfileID: target.ProfileID})

	rr := httptest.NewRecorder()
	h.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusNotFound)
	}
	if got := storage.keys(); len(got) != 0 {
		t.Errorf("requestedKeys = %v, want empty", got)
	}
}

// TestShow_WhenExportsUnavailable closes the request on a deployment that has no
// object storage, instead of describing a feature it cannot serve.
//
// [Ja] TestShow_WhenExportsUnavailable は、オブジェクトストレージを持たないデプロイで
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
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusServiceUnavailable)
	}
	if got := rr.Header().Get("Content-Disposition"); got != "" {
		t.Errorf("Content-Disposition = %q, want empty", got)
	}
}

// TestShow_WithoutSignedInIdentity fails the request when the route was wired
// without RequireAuth, rather than opening an archive for an unknown owner.
//
// [Ja] TestShow_WithoutSignedInIdentity は、RequireAuth 無しでルートが登録された
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
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusInternalServerError)
	}
	if got := storage.keys(); len(got) != 0 {
		t.Errorf("requestedKeys = %v, want empty", got)
	}
}

// newServerChain wraps the handler in the middleware the download route runs
// under, with the signed-in identity RequireAuth would have resolved.
// PrivateCache is included because it sets a Cache-Control the handler has to
// override for a downloaded file.
//
// [Ja] newServerChain はダウンロードのルートが動くミドルウェアでハンドラーを包み、
// RequireAuth が解決したはずのログイン中の identity を載せる。PrivateCache を
// 含めるのは、ダウンロードされるファイルのためにハンドラーが上書きしなければならない
// Cache-Control を設定するミドルウェアであるため。
func newServerChain(t *testing.T, tx *sql.Tx, storage usecase.ExportObjectStorage, owner testutil.ProfileOwner) http.Handler {
	t.Helper()

	h := newHandler(t, tx, storage, true)
	return middleware.PrivateCache(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.Show(w, r.WithContext(signedInContext(r.Context(), owner)))
	}))
}

// TestShow_ThroughReverseProxy sends the download through a reverse proxy, the
// shape the request has in every environment: a proxy sits in front of the app
// in development and in production alike. What the reader ends up with must be
// the same archive, described by the same headers, after that hop.
//
// [Ja] TestShow_ThroughReverseProxy はダウンロードをリバースプロキシ越しに送る。
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
		t.Fatalf("オリジンの URL の解析に失敗: %v", err)
	}
	proxy := httptest.NewServer(httputil.NewSingleHostReverseProxy(originURL))
	defer proxy.Close()

	resp, err := http.Get(proxy.URL + "/settings/export/download")
	if err != nil {
		t.Fatalf("リクエストの送信に失敗: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ステータスコードが不正: got %v, want %v", resp.StatusCode, http.StatusOK)
	}
	assertArchiveHeaders(t, resp.Header, len(archive))

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("レスポンスの読み取りに失敗: %v", err)
	}
	if !bytes.Equal(body, archive) {
		t.Errorf("レスポンスのバイト列がアーカイブと一致しません (got %d bytes, want %d bytes)", len(body), len(archive))
	}
	assertReadableArchive(t, body)

	if handedOver := storage.handedOver(); handedOver == nil || !handedOver.isClosed() {
		t.Error("ハンドラーがストリームを閉じていません")
	}
}
