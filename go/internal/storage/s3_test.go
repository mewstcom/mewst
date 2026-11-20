package storage_test

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mewstcom/mewst/go/internal/storage"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// S3ExportStorageはApplication層のportを満たさなければならない。
var _ usecase.ExportObjectStorage = (*storage.S3ExportStorage)(nil)

const testBucket = "test-bucket"

// fakeS3はアダプタが発行する操作 (PutObject, CreateMultipartUpload,
// UploadPart, CompleteMultipartUpload, AbortMultipartUpload, GetObject,
// DeleteObject, ListObjectsV2) だけを実装した最小のパス形式S3サーバー。
type fakeS3 struct {
	mu               sync.Mutex
	objects          map[string][]byte
	objectModifiedAt map[string]time.Time
	putHeaders       map[string]http.Header
	parts            map[int][]byte
	multipartCreated bool
	completed        bool
	aborted          bool
	deleted          []string
	deleteErrorCode  string
	listCalls        int

	// listStartAftersは各ListObjectsV2リクエストのstart-afterを記録し、
	// どのリクエストが再開位置を運んだかをテストが判別できるようにする。
	listStartAfters []string

	// listPageSizeはListObjectsV2の1ページあたりのキー数の上限。
	// オブジェクト数より小さい値にすると、アダプタはcontinuation tokenに
	// よるページング経路を通る。0は上限なし。
	listPageSize int

	// listOmitNextTokenは切り詰められたListObjectsV2ページから
	// NextContinuationTokenを省き、listOmitLastModifiedは各Contentsから
	// LastModifiedを省く。いずれもアダプタが無限ループやゼロ値の時刻を
	// yieldせずに拒否すべき不正な応答を作る。
	listOmitNextToken    bool
	listOmitLastModified bool

	// blockPut / blockPartsはcontextキャンセルのテスト用に、PutObject /
	// UploadPartをクライアント切断かunblockのcloseまでブロックさせる。
	// partStartedは最初にブロックしたUploadPartの到達時にcloseされ、
	// テストがmultipart uploadの開始後にだけキャンセルできるようにする。
	blockPut        bool
	blockParts      bool
	unblock         chan struct{}
	partStarted     chan struct{}
	partStartedOnce sync.Once
}

func newFakeS3() *fakeS3 {
	return &fakeS3{
		objects:          make(map[string][]byte),
		objectModifiedAt: make(map[string]time.Time),
		putHeaders:       make(map[string]http.Header),
		parts:            make(map[int][]byte),
		unblock:          make(chan struct{}),
		partStarted:      make(chan struct{}),
	}
}

// waitForClientOrUnblockはクライアント切断かunblockのcloseまで
// ハンドラーを停止させる。呼び出し側は先にリクエストボディを読み切ること。
// net/httpはボディを消費し終えた後にだけクライアント切断の監視を開始する
// ため、ボディが未読のままだとリクエストcontextが永遠に生き続ける。unblock
// はテストサーバーを確実に終了させるための保険。
func (f *fakeS3) waitForClientOrUnblock(r *http.Request) {
	select {
	case <-r.Context().Done():
	case <-f.unblock:
	}
}

func (f *fakeS3) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.URL.Path, "/"+testBucket+"/")
		q := r.URL.Query()

		switch {
		case r.Method == http.MethodPost && q.Has("uploads"):
			f.mu.Lock()
			f.multipartCreated = true
			f.mu.Unlock()
			w.Header().Set("Content-Type", "application/xml")
			_, _ = fmt.Fprintf(w, `<InitiateMultipartUploadResult><Bucket>%s</Bucket><Key>%s</Key><UploadId>test-upload-id</UploadId></InitiateMultipartUploadResult>`, testBucket, key)

		case r.Method == http.MethodPut && q.Get("partNumber") != "":
			if f.blockParts {
				_, _ = io.Copy(io.Discard, r.Body)
				f.partStartedOnce.Do(func() { close(f.partStarted) })
				f.waitForClientOrUnblock(r)
				return
			}
			partNumber, err := strconv.Atoi(q.Get("partNumber"))
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			f.mu.Lock()
			f.parts[partNumber] = body
			f.mu.Unlock()
			w.Header().Set("ETag", fmt.Sprintf("%q", "etag-"+q.Get("partNumber")))

		case r.Method == http.MethodPost && q.Get("uploadId") != "":
			// CompleteMultipartUpload: パート番号順に連結して最終
			// オブジェクトにする。
			f.mu.Lock()
			numbers := make([]int, 0, len(f.parts))
			for n := range f.parts {
				numbers = append(numbers, n)
			}
			sort.Ints(numbers)
			var buf bytes.Buffer
			for _, n := range numbers {
				buf.Write(f.parts[n])
			}
			f.objects[key] = buf.Bytes()
			f.objectModifiedAt[key] = time.Now().UTC()
			f.completed = true
			f.mu.Unlock()
			w.Header().Set("Content-Type", "application/xml")
			_, _ = fmt.Fprintf(w, `<CompleteMultipartUploadResult><Bucket>%s</Bucket><Key>%s</Key><ETag>"etag"</ETag></CompleteMultipartUploadResult>`, testBucket, key)

		case r.Method == http.MethodDelete && q.Get("uploadId") != "":
			f.mu.Lock()
			f.aborted = true
			f.mu.Unlock()
			w.WriteHeader(http.StatusNoContent)

		case r.Method == http.MethodDelete:
			// 実際のS3は存在しないキーにも204を返すが、fakeはあえて
			// 404 NoSuchKeyを返し、アダプタの「404は成功扱い」経路を通す。
			if f.deleteErrorCode != "" {
				w.Header().Set("Content-Type", "application/xml")
				w.WriteHeader(http.StatusNotFound)
				_, _ = fmt.Fprintf(w, "<Error><Code>%s</Code><Message>forced delete error</Message></Error>", f.deleteErrorCode)
				return
			}

			f.mu.Lock()
			_, exists := f.objects[key]
			if exists {
				delete(f.objects, key)
				delete(f.objectModifiedAt, key)
				f.deleted = append(f.deleted, key)
			}
			f.mu.Unlock()
			if !exists {
				w.Header().Set("Content-Type", "application/xml")
				w.WriteHeader(http.StatusNotFound)
				_, _ = io.WriteString(w, `<Error><Code>NoSuchKey</Code><Message>The specified key does not exist.</Message></Error>`)
				return
			}
			w.WriteHeader(http.StatusNoContent)

		case r.Method == http.MethodGet && q.Get("list-type") == "2":
			// ListObjectsV2: prefixに一致するキーを辞書順で返し、
			// listPageSize単位でページングする。continuation tokenは前ページ
			// の最後のキー。
			f.mu.Lock()
			f.listCalls++
			f.listStartAfters = append(f.listStartAfters, q.Get("start-after"))
			keys := make([]string, 0, len(f.objects))
			modifiedAtByKey := make(map[string]time.Time, len(f.objects))
			for k := range f.objects {
				if strings.HasPrefix(k, q.Get("prefix")) {
					keys = append(keys, k)
					modifiedAtByKey[k] = f.objectModifiedAt[k]
				}
			}
			f.mu.Unlock()
			sort.Strings(keys)

			// continuation tokenを持つリクエストはtokenの位置から再開する。
			// start-afterが効くのはtokenを持たないリクエストだけ。
			start := 0
			if resumeAfter := cmp.Or(q.Get("continuation-token"), q.Get("start-after")); resumeAfter != "" {
				start = sort.SearchStrings(keys, resumeAfter)
				if start < len(keys) && keys[start] == resumeAfter {
					start++
				}
			}
			end := len(keys)
			if f.listPageSize > 0 && start+f.listPageSize < end {
				end = start + f.listPageSize
			}
			page := keys[start:end]
			truncated := end < len(keys)

			w.Header().Set("Content-Type", "application/xml")
			var b strings.Builder
			b.WriteString("<ListBucketResult>")
			fmt.Fprintf(&b, "<IsTruncated>%t</IsTruncated>", truncated)
			for _, k := range page {
				if f.listOmitLastModified {
					fmt.Fprintf(&b, "<Contents><Key>%s</Key></Contents>", k)
					continue
				}
				fmt.Fprintf(&b, "<Contents><Key>%s</Key><LastModified>%s</LastModified></Contents>", k, modifiedAtByKey[k].Format(time.RFC3339))
			}
			if truncated && !f.listOmitNextToken {
				fmt.Fprintf(&b, "<NextContinuationToken>%s</NextContinuationToken>", page[len(page)-1])
			}
			b.WriteString("</ListBucketResult>")
			_, _ = io.WriteString(w, b.String())

		case r.Method == http.MethodPut:
			if f.blockPut {
				_, _ = io.Copy(io.Discard, r.Body)
				f.waitForClientOrUnblock(r)
				return
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			f.mu.Lock()
			f.objects[key] = body
			f.objectModifiedAt[key] = time.Now().UTC()
			f.putHeaders[key] = r.Header.Clone()
			f.mu.Unlock()
			w.Header().Set("ETag", `"etag"`)

		case r.Method == http.MethodGet:
			f.mu.Lock()
			body, ok := f.objects[key]
			f.mu.Unlock()
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/zip")
			w.Header().Set("Content-Length", strconv.Itoa(len(body)))
			_, _ = w.Write(body)

		default:
			w.WriteHeader(http.StatusNotImplemented)
		}
	})
}

func newTestStorage(t *testing.T, f *fakeS3) *storage.S3ExportStorage {
	t.Helper()

	srv := httptest.NewServer(f.handler())
	t.Cleanup(srv.Close)

	return storage.NewS3ExportStorage(storage.S3Config{
		BucketName:      testBucket,
		Endpoint:        srv.URL,
		AccessKeyID:     "test-access-key-id",
		SecretAccessKey: "test-secret-access-key",
		Region:          "auto",
	})
}

func TestS3ExportStorage_Upload(t *testing.T) {
	t.Parallel()

	f := newFakeS3()
	st := newTestStorage(t, f)
	body := []byte("zip-bytes")
	key := "exports/profile-id/export-id.zip"

	if err := st.Upload(context.Background(), key, bytes.NewReader(body)); err != nil {
		t.Fatalf("Upload()のエラー = %v", err)
	}

	f.mu.Lock()
	stored := f.objects[key]
	headers := f.putHeaders[key]
	f.mu.Unlock()

	if !bytes.Equal(stored, body) {
		t.Errorf("保存したオブジェクト = %q、期待値 = %q", stored, body)
	}
	if got := headers.Get("Content-Type"); got != "application/zip" {
		t.Errorf("Content-Type = %q、期待値 = %q", got, "application/zip")
	}
	if got := headers.Get("Cache-Control"); got != "private, no-store" {
		t.Errorf("Cache-Control = %q、期待値 = %q", got, "private, no-store")
	}
}

func TestS3ExportStorage_Upload_MultipartStreaming(t *testing.T) {
	t.Parallel()

	f := newFakeS3()
	st := newTestStorage(t, f)

	// 6 MiBは既定パートサイズ5 MiBを超えるためmultipart経路
	// (2パート) を通る。実際の生成処理と同じくio.Pipeでストリーミングする。
	content := deterministicBytes(6 << 20)
	pr, pw := io.Pipe()
	go func() {
		_, err := pw.Write(content)
		_ = pw.CloseWithError(err)
	}()

	key := "exports/profile-id/export-id.zip"
	if err := st.Upload(context.Background(), key, pr); err != nil {
		t.Fatalf("Upload()のエラー = %v", err)
	}

	f.mu.Lock()
	created := f.multipartCreated
	completed := f.completed
	partCount := len(f.parts)
	stored := f.objects[key]
	f.mu.Unlock()

	if !created {
		t.Error("CreateMultipartUploadが呼ばれていない")
	}
	if !completed {
		t.Error("CompleteMultipartUploadが呼ばれていない")
	}
	if partCount < 2 {
		t.Errorf("パートの件数 = %d、期待値 = 2以上", partCount)
	}
	if !bytes.Equal(stored, content) {
		t.Errorf("保存したオブジェクトの長さ = %d、期待値 = %d (内容が一致しない)", len(stored), len(content))
	}
}

func TestS3ExportStorage_Upload_ProducerErrorAbortsMultipart(t *testing.T) {
	t.Parallel()

	f := newFakeS3()
	st := newTestStorage(t, f)

	// 最初のパートの後でproducerを失敗させ、生成途中でエラーになる
	// アーカイブbuilderを模す。アダプタは開始済みのmultipart uploadを
	// 中断し、producerのエラーを伝搬しなければならない。
	producerErr := errors.New("archive build failed")
	pr, pw := io.Pipe()
	go func() {
		if _, err := pw.Write(deterministicBytes(5<<20 + 1024)); err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		_ = pw.CloseWithError(producerErr)
	}()

	err := st.Upload(context.Background(), "exports/profile-id/export-id.zip", pr)
	if err == nil {
		t.Fatal("Upload()のエラー = nil、生成側のエラーを期待")
	}
	if !strings.Contains(err.Error(), producerErr.Error()) {
		t.Errorf("Upload()のエラー = %v、%qを含むことを期待", err, producerErr.Error())
	}

	f.mu.Lock()
	aborted := f.aborted
	completed := f.completed
	f.mu.Unlock()

	if !aborted {
		t.Error("AbortMultipartUploadが呼ばれていない")
	}
	if completed {
		t.Error("失敗したアップロードでCompleteMultipartUploadが呼ばれた")
	}
}

func TestS3ExportStorage_Upload_ContextCancellation(t *testing.T) {
	t.Parallel()

	f := newFakeS3()
	f.blockPut = true
	st := newTestStorage(t, f)
	// newTestStorageの後に登録することで (cleanupはLIFO順)、サーバーの
	// Closeより先に実行され、ブロックしたままのハンドラーを解放する。
	t.Cleanup(func() { close(f.unblock) })

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		// fakeサーバーがPutObjectリクエストを保持している間に
		// キャンセルする。
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := st.Upload(ctx, "exports/profile-id/export-id.zip", bytes.NewReader([]byte("zip-bytes")))
	if err == nil {
		t.Fatal("Upload()のエラー = nil、コンテキストのキャンセルによるエラーを期待")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Upload()のエラー = %v、errors.Is(err, context.Canceled) を満たすエラーを期待", err)
	}

	// abortMultipartUploadのno-op契約を固定する: 単発PutObject経路
	// ではmultipart uploadを開始も中断もしない。
	f.mu.Lock()
	created := f.multipartCreated
	aborted := f.aborted
	f.mu.Unlock()

	if created {
		t.Error("1回で送る小さなアップロードでCreateMultipartUploadが呼ばれた")
	}
	if aborted {
		t.Error("マルチパートではない失敗でAbortMultipartUploadが呼ばれた")
	}
}

func TestS3ExportStorage_Upload_ContextCancellationAbortsMultipart(t *testing.T) {
	t.Parallel()

	f := newFakeS3()
	f.blockParts = true
	st := newTestStorage(t, f)
	t.Cleanup(func() { close(f.unblock) })

	// 複数パート分をストリーミングしてmultipart uploadを開始させ、fake
	// サーバーが最初のUploadPartを受信した後にだけキャンセルする。アダプタは
	// 開始済みのmultipart uploadをそれでも中断しなければならない (cleanup
	// contextはキャンセル済みのupload contextから切り離されている)。
	pr, pw := io.Pipe()
	go func() {
		_, err := pw.Write(deterministicBytes(6 << 20))
		_ = pw.CloseWithError(err)
	}()
	// アップローダーが途中で読むのをやめた場合に備え、writer goroutineの
	// ブロックを解除する。
	t.Cleanup(func() { _ = pr.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		<-f.partStarted
		cancel()
	}()

	err := st.Upload(ctx, "exports/profile-id/export-id.zip", pr)
	if err == nil {
		t.Fatal("Upload()のエラー = nil、コンテキストのキャンセルによるエラーを期待")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Upload()のエラー = %v、errors.Is(err, context.Canceled) を満たすエラーを期待", err)
	}

	f.mu.Lock()
	created := f.multipartCreated
	aborted := f.aborted
	completed := f.completed
	f.mu.Unlock()

	if !created {
		t.Error("CreateMultipartUploadが呼ばれていない")
	}
	if !aborted {
		t.Error("コンテキストのキャンセル後にAbortMultipartUploadが呼ばれていない")
	}
	if completed {
		t.Error("キャンセルしたアップロードでCompleteMultipartUploadが呼ばれた")
	}
}

func TestS3ExportStorage_Download(t *testing.T) {
	t.Parallel()

	f := newFakeS3()
	content := deterministicBytes(64 << 10)
	key := "exports/profile-id/export-id.zip"
	f.objects[key] = content
	st := newTestStorage(t, f)

	body, size, err := st.Download(context.Background(), key)
	if err != nil {
		t.Fatalf("Download()のエラー = %v", err)
	}
	defer func() { _ = body.Close() }()

	if size != int64(len(content)) {
		t.Errorf("size = %d、期待値 = %d", size, len(content))
	}
	got, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("本文の読み込みに失敗: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("本文の長さ = %d、期待値 = %d (内容が一致しない)", len(got), len(content))
	}
	if err := body.Close(); err != nil {
		t.Errorf("Close()のエラー = %v", err)
	}
}

func TestS3ExportStorage_Download_CloseBeforeEOF(t *testing.T) {
	t.Parallel()

	f := newFakeS3()
	key := "exports/profile-id/export-id.zip"
	f.objects[key] = deterministicBytes(1 << 20)
	st := newTestStorage(t, f)

	body, _, err := st.Download(context.Background(), key)
	if err != nil {
		t.Fatalf("Download()のエラー = %v", err)
	}

	// クライアント切断ではダウンロード途中でストリームを閉じる。
	// 返されたbodyはハングせずに背後のHTTPレスポンスを解放しなければ
	// ならない。
	buf := make([]byte, 1024)
	if _, err := io.ReadFull(body, buf); err != nil {
		t.Fatalf("最初のチャンクの読み込みに失敗: %v", err)
	}
	if err := body.Close(); err != nil {
		t.Fatalf("Close()のエラー = %v", err)
	}
	if _, err := body.Read(buf); err == nil {
		t.Error("Close()後のRead()のエラー = nil、エラーを期待")
	}
}

func TestS3ExportStorage_Delete(t *testing.T) {
	t.Parallel()

	f := newFakeS3()
	key := "exports/profile-id/export-id.zip"
	otherKey := "exports/profile-id/other-export-id.zip"
	f.objects[key] = []byte("zip-bytes")
	f.objects[otherKey] = []byte("other-zip-bytes")
	st := newTestStorage(t, f)

	if err := st.Delete(context.Background(), key); err != nil {
		t.Fatalf("Delete()のエラー = %v", err)
	}

	f.mu.Lock()
	_, exists := f.objects[key]
	_, otherExists := f.objects[otherKey]
	deleted := append([]string(nil), f.deleted...)
	f.mu.Unlock()

	if exists {
		t.Error("Delete()の後もオブジェクトが残っている")
	}
	if !otherExists {
		t.Error("Delete()が無関係なオブジェクトを削除した")
	}
	if len(deleted) != 1 || deleted[0] != key {
		t.Errorf("削除したキー = %v、期待値 = [%s]", deleted, key)
	}
}

func TestS3ExportStorage_Delete_MissingKeyIsSuccess(t *testing.T) {
	t.Parallel()

	f := newFakeS3()
	st := newTestStorage(t, f)

	// fakeは存在しないキーに404 NoSuchKeyを返す。リトライされる
	// cleanupジョブが冪等であるよう、アダプタはこれを成功として扱わなければ
	// ならない。
	if err := st.Delete(context.Background(), "exports/profile-id/missing.zip"); err != nil {
		t.Fatalf("存在しないキーでDelete()のエラー = %v、期待値 = nil", err)
	}
}

func TestS3ExportStorage_Delete_NoSuchBucketIsError(t *testing.T) {
	t.Parallel()

	f := newFakeS3()
	f.deleteErrorCode = "NoSuchBucket"
	st := newTestStorage(t, f)

	err := st.Delete(context.Background(), "exports/profile-id/export-id.zip")
	if err == nil {
		t.Fatal("Delete()のエラー = nil、NoSuchBucketのエラーを期待")
	}
	if !strings.Contains(err.Error(), "NoSuchBucket") {
		t.Errorf("Delete()のエラー = %v、NoSuchBucketを期待", err)
	}
}

func TestS3ExportStorage_ListPrefix(t *testing.T) {
	t.Parallel()

	f := newFakeS3()
	// ページングを強制する: 一致する5オブジェクトを1ページ2キーで
	// 返すと、ListObjectsV2は3回呼ばれる。
	f.listPageSize = 2
	exportKeys := []string{
		"exports/profile-a/export-1.zip",
		"exports/profile-a/export-2.zip",
		"exports/profile-b/export-3.zip",
		"exports/profile-c/export-4.zip",
		"exports/profile-c/export-5.zip",
	}
	wantModifiedAt := make(map[string]time.Time, len(exportKeys))
	for i, k := range exportKeys {
		f.objects[k] = []byte("zip-bytes")
		modifiedAt := time.Date(2026, time.July, 23, 12, i, 0, 0, time.UTC)
		f.objectModifiedAt[k] = modifiedAt
		wantModifiedAt[k] = modifiedAt
	}
	// バケットを共有する他機能のオブジェクトは一覧されてはならない。
	f.objects["images/profile-a/avatar.png"] = []byte("png-bytes")
	st := newTestStorage(t, f)

	gotModifiedAt := make(map[string]time.Time, len(exportKeys))
	err := st.ListPrefix(context.Background(), "exports/", "", func(key string, lastModified time.Time) error {
		gotModifiedAt[key] = lastModified
		return nil
	})
	if err != nil {
		t.Fatalf("ListPrefix()のエラー = %v", err)
	}

	if !maps.Equal(gotModifiedAt, wantModifiedAt) {
		t.Errorf("ListPrefix() = %v、期待値 = %v", gotModifiedAt, wantModifiedAt)
	}

	f.mu.Lock()
	listCalls := f.listCalls
	f.mu.Unlock()
	if listCalls != 3 {
		t.Errorf("ListObjectsV2の呼び出し回数 = %d、期待値 = 3 (ページングを辿っていない)", listCalls)
	}
}

// TestS3ExportStorage_ListPrefix_StartAfterは再開位置を固定する。走査が有界な
// 呼び出し側は止まったキーを次の走査へ渡すため、一覧はそのキーより厳密に後ろから始まり、
// 続くページングはそこへ戻らず前進し続ける必要がある。
func TestS3ExportStorage_ListPrefix_StartAfter(t *testing.T) {
	t.Parallel()

	f := newFakeS3()
	// 1ページ2キーにして、再開したあと少なくとも1回はページングさせる。
	f.listPageSize = 2
	for i, k := range []string{
		"exports/profile-a/export-1.zip",
		"exports/profile-a/export-2.zip",
		"exports/profile-b/export-3.zip",
		"exports/profile-c/export-4.zip",
		"exports/profile-c/export-5.zip",
	} {
		f.objects[k] = []byte("zip-bytes")
		f.objectModifiedAt[k] = time.Date(2026, time.July, 23, 12, i, 0, 0, time.UTC)
	}
	st := newTestStorage(t, f)

	var got []string
	err := st.ListPrefix(context.Background(), "exports/", "exports/profile-a/export-2.zip", func(key string, _ time.Time) error {
		got = append(got, key)
		return nil
	})
	if err != nil {
		t.Fatalf("ListPrefix()のエラー = %v", err)
	}

	want := []string{
		"exports/profile-b/export-3.zip",
		"exports/profile-c/export-4.zip",
		"exports/profile-c/export-5.zip",
	}
	if !slices.Equal(got, want) {
		t.Errorf("ListPrefix() = %v、期待値 = %v", got, want)
	}

	// 再開位置を運ぶのは最初のリクエストだけである。以降はcontinuation tokenの
	// 位置から再開し、tokenはすでに先の位置を表しているため、元のキーを再送しても走査を
	// 後ろへ引き戻すことしかできない。
	f.mu.Lock()
	gotStartAfters := slices.Clone(f.listStartAfters)
	f.mu.Unlock()

	wantStartAfters := []string{"exports/profile-a/export-2.zip", ""}
	if !slices.Equal(gotStartAfters, wantStartAfters) {
		t.Errorf("各ListObjectsV2のstart-after = %v、期待値 = %v", gotStartAfters, wantStartAfters)
	}
}

func TestS3ExportStorage_ListPrefix_Empty(t *testing.T) {
	t.Parallel()

	f := newFakeS3()
	st := newTestStorage(t, f)

	var keys []string
	err := st.ListPrefix(context.Background(), "exports/", "", func(key string, _ time.Time) error {
		keys = append(keys, key)
		return nil
	})
	if err != nil {
		t.Fatalf("ListPrefix()のエラー = %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("ListPrefix() = %v、空を期待", keys)
	}
}

func TestS3ExportStorage_ListPrefix_YieldError(t *testing.T) {
	t.Parallel()

	f := newFakeS3()
	key := "exports/profile-id/export-id.zip"
	f.objects[key] = []byte("zip-bytes")
	f.objectModifiedAt[key] = time.Date(2026, time.July, 23, 12, 0, 0, 0, time.UTC)
	st := newTestStorage(t, f)
	wantErr := errors.New("stop listing")

	err := st.ListPrefix(context.Background(), "exports/", "", func(string, time.Time) error {
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Errorf("ListPrefix()のエラー = %v、errors.Is(err, wantErr) を満たすエラーを期待", err)
	}
}

func TestS3ExportStorage_ListPrefix_TruncatedWithoutTokenIsError(t *testing.T) {
	t.Parallel()

	f := newFakeS3()
	// 1ページ1キーで一致オブジェクトが2件だと最初のページが切り詰め
	// られ、tokenを省くとアダプタは次へ進めない。1ページ目を永久に取り直さず
	// 失敗しなければならない。
	f.listPageSize = 1
	f.listOmitNextToken = true
	f.objects["exports/profile-a/export-1.zip"] = []byte("zip-bytes")
	f.objects["exports/profile-a/export-2.zip"] = []byte("zip-bytes")
	st := newTestStorage(t, f)

	err := st.ListPrefix(context.Background(), "exports/", "", func(string, time.Time) error {
		return nil
	})
	if err == nil {
		t.Fatal("継続トークンの無い途中のページでListPrefix()のエラー = nil、エラーを期待")
	}

	f.mu.Lock()
	listCalls := f.listCalls
	f.mu.Unlock()
	if listCalls != 1 {
		t.Errorf("ListObjectsV2の呼び出し回数 = %d、期待値 = 1 (アダプタが最初のページを繰り返した)", listCalls)
	}
}

func TestS3ExportStorage_ListPrefix_MissingLastModifiedIsError(t *testing.T) {
	t.Parallel()

	f := newFakeS3()
	// LastModifiedの無いContentsは猶予期間を判定する時刻を呼び出し側に
	// 渡せないため、アダプタはゼロ値の時刻をyieldせず拒否しなければならない。
	f.listOmitLastModified = true
	f.objects["exports/profile-a/export-1.zip"] = []byte("zip-bytes")
	st := newTestStorage(t, f)

	err := st.ListPrefix(context.Background(), "exports/", "", func(string, time.Time) error {
		return nil
	})
	if err == nil {
		t.Fatal("LastModifiedの無いContentsの要素でListPrefix()のエラー = nil、エラーを期待")
	}
}

// deterministicBytesは決定的なパターンのsizeバイトを返す。内容比較で
// 並び替えや欠落を検出できるようにする。
func deterministicBytes(size int) []byte {
	b := make([]byte, size)
	for i := range b {
		b[i] = byte(i % 251)
	}
	return b
}
