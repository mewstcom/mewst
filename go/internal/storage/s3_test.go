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

// S3ExportStorage must satisfy the Application-layer port.
//
// [Ja] S3ExportStorage は Application 層の port を満たさなければならない。
var _ usecase.ExportObjectStorage = (*storage.S3ExportStorage)(nil)

const testBucket = "test-bucket"

// fakeS3 is a minimal path-style S3 server covering only the operations the
// adapter issues: PutObject, CreateMultipartUpload, UploadPart,
// CompleteMultipartUpload, AbortMultipartUpload, GetObject, DeleteObject and
// ListObjectsV2.
//
// [Ja] fakeS3 はアダプタが発行する操作 (PutObject, CreateMultipartUpload,
// UploadPart, CompleteMultipartUpload, AbortMultipartUpload, GetObject,
// DeleteObject, ListObjectsV2) だけを実装した最小のパス形式 S3 サーバー。
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

	// listStartAfters records the start-after parameter of every ListObjectsV2
	// request, so a test can tell which of them carried a resume position.
	//
	// [Ja] listStartAfters は各 ListObjectsV2 リクエストの start-after を記録し、
	// どのリクエストが再開位置を運んだかをテストが判別できるようにする。
	listStartAfters []string

	// listPageSize caps the number of keys per ListObjectsV2 page. A value
	// smaller than the object count forces the adapter through the
	// continuation-token pagination path. Zero means no cap.
	//
	// [Ja] listPageSize は ListObjectsV2 の 1 ページあたりのキー数の上限。
	// オブジェクト数より小さい値にすると、アダプタは continuation token に
	// よるページング経路を通る。0 は上限なし。
	listPageSize int

	// listOmitNextToken makes a truncated ListObjectsV2 page omit the
	// NextContinuationToken, and listOmitLastModified makes each Contents
	// entry omit LastModified. Both produce the malformed responses the
	// adapter must reject instead of looping forever or yielding a zero time.
	//
	// [Ja] listOmitNextToken は切り詰められた ListObjectsV2 ページから
	// NextContinuationToken を省き、listOmitLastModified は各 Contents から
	// LastModified を省く。いずれもアダプタが無限ループやゼロ値の時刻を
	// yield せずに拒否すべき不正な応答を作る。
	listOmitNextToken    bool
	listOmitLastModified bool

	// blockPut / blockParts make PutObject / UploadPart hang until the client
	// disconnects or unblock is closed, for the context-cancellation tests.
	// partStarted is closed when the first blocked UploadPart arrives so a
	// test can cancel only after the multipart upload has actually started.
	//
	// [Ja] blockPut / blockParts は context キャンセルのテスト用に、PutObject /
	// UploadPart をクライアント切断か unblock の close までブロックさせる。
	// partStarted は最初にブロックした UploadPart の到達時に close され、
	// テストが multipart upload の開始後にだけキャンセルできるようにする。
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

// waitForClientOrUnblock parks the handler until the client disconnects or
// unblock is closed. Callers must drain the request body first: net/http
// starts watching for a client disconnect only after the body is consumed,
// so an unread body would keep the request context alive forever. unblock is
// the fallback so the test server can always shut down.
//
// [Ja] waitForClientOrUnblock はクライアント切断か unblock の close まで
// ハンドラーを停止させる。呼び出し側は先にリクエストボディを読み切ること。
// net/http はボディを消費し終えた後にだけクライアント切断の監視を開始する
// ため、ボディが未読のままだとリクエスト context が永遠に生き続ける。unblock
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
			// CompleteMultipartUpload: concatenate the parts in part-number
			// order into the final object.
			//
			// [Ja] CompleteMultipartUpload: パート番号順に連結して最終
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
			// Real S3 answers 204 even for a missing key, but the fake returns
			// 404 NoSuchKey instead so the adapter's 404-as-success path is
			// exercised.
			//
			// [Ja] 実際の S3 は存在しないキーにも 204 を返すが、fake はあえて
			// 404 NoSuchKey を返し、アダプタの「404 は成功扱い」経路を通す。
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
			// ListObjectsV2: return the prefix-matched keys in lexicographic
			// order, paginated by listPageSize. The continuation token is the
			// last key of the previous page.
			//
			// [Ja] ListObjectsV2: prefix に一致するキーを辞書順で返し、
			// listPageSize 単位でページングする。continuation token は前ページ
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

			// A request carrying a continuation token resumes from the token;
			// start-after applies only to a request without one.
			//
			// [Ja] continuation token を持つリクエストは token の位置から再開する。
			// start-after が効くのは token を持たないリクエストだけ。
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
		t.Fatalf("Upload() error = %v", err)
	}

	f.mu.Lock()
	stored := f.objects[key]
	headers := f.putHeaders[key]
	f.mu.Unlock()

	if !bytes.Equal(stored, body) {
		t.Errorf("stored object = %q, want %q", stored, body)
	}
	if got := headers.Get("Content-Type"); got != "application/zip" {
		t.Errorf("Content-Type = %q, want %q", got, "application/zip")
	}
	if got := headers.Get("Cache-Control"); got != "private, no-store" {
		t.Errorf("Cache-Control = %q, want %q", got, "private, no-store")
	}
}

func TestS3ExportStorage_Upload_MultipartStreaming(t *testing.T) {
	t.Parallel()

	f := newFakeS3()
	st := newTestStorage(t, f)

	// 6 MiB exceeds the 5 MiB default part size, forcing the multipart path
	// (2 parts) while streaming through an io.Pipe like the real generator.
	//
	// [Ja] 6 MiB は既定パートサイズ 5 MiB を超えるため multipart 経路
	// (2 パート) を通る。実際の生成処理と同じく io.Pipe でストリーミングする。
	content := deterministicBytes(6 << 20)
	pr, pw := io.Pipe()
	go func() {
		_, err := pw.Write(content)
		_ = pw.CloseWithError(err)
	}()

	key := "exports/profile-id/export-id.zip"
	if err := st.Upload(context.Background(), key, pr); err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	f.mu.Lock()
	created := f.multipartCreated
	completed := f.completed
	partCount := len(f.parts)
	stored := f.objects[key]
	f.mu.Unlock()

	if !created {
		t.Error("CreateMultipartUpload was not called")
	}
	if !completed {
		t.Error("CompleteMultipartUpload was not called")
	}
	if partCount < 2 {
		t.Errorf("part count = %d, want >= 2", partCount)
	}
	if !bytes.Equal(stored, content) {
		t.Errorf("stored object length = %d, want %d (content mismatch)", len(stored), len(content))
	}
}

func TestS3ExportStorage_Upload_ProducerErrorAbortsMultipart(t *testing.T) {
	t.Parallel()

	f := newFakeS3()
	st := newTestStorage(t, f)

	// The producer fails after the first part, mimicking an archive builder
	// that errors mid-generation. The adapter must abort the started
	// multipart upload and propagate the producer error.
	//
	// [Ja] 最初のパートの後で producer を失敗させ、生成途中でエラーになる
	// アーカイブ builder を模す。アダプタは開始済みの multipart upload を
	// 中断し、producer のエラーを伝搬しなければならない。
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
		t.Fatal("Upload() error = nil, want producer error")
	}
	if !strings.Contains(err.Error(), producerErr.Error()) {
		t.Errorf("Upload() error = %v, want to contain %q", err, producerErr.Error())
	}

	f.mu.Lock()
	aborted := f.aborted
	completed := f.completed
	f.mu.Unlock()

	if !aborted {
		t.Error("AbortMultipartUpload was not called")
	}
	if completed {
		t.Error("CompleteMultipartUpload was called for a failed upload")
	}
}

func TestS3ExportStorage_Upload_ContextCancellation(t *testing.T) {
	t.Parallel()

	f := newFakeS3()
	f.blockPut = true
	st := newTestStorage(t, f)
	// Registered after newTestStorage, so this runs before the server Close
	// (cleanups run in LIFO order) and releases a still-blocked handler.
	//
	// [Ja] newTestStorage の後に登録することで (cleanup は LIFO 順)、サーバーの
	// Close より先に実行され、ブロックしたままのハンドラーを解放する。
	t.Cleanup(func() { close(f.unblock) })

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		// Cancel while the fake server is holding the PutObject request open.
		//
		// [Ja] fake サーバーが PutObject リクエストを保持している間に
		// キャンセルする。
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := st.Upload(ctx, "exports/profile-id/export-id.zip", bytes.NewReader([]byte("zip-bytes")))
	if err == nil {
		t.Fatal("Upload() error = nil, want context cancellation error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Upload() error = %v, want errors.Is(err, context.Canceled)", err)
	}

	// Pin the no-op contract of abortMultipartUpload: the single PutObject
	// path must neither start nor abort a multipart upload.
	//
	// [Ja] abortMultipartUpload の no-op 契約を固定する: 単発 PutObject 経路
	// では multipart upload を開始も中断もしない。
	f.mu.Lock()
	created := f.multipartCreated
	aborted := f.aborted
	f.mu.Unlock()

	if created {
		t.Error("CreateMultipartUpload was called for a small single-put upload")
	}
	if aborted {
		t.Error("AbortMultipartUpload was called for a non-multipart failure")
	}
}

func TestS3ExportStorage_Upload_ContextCancellationAbortsMultipart(t *testing.T) {
	t.Parallel()

	f := newFakeS3()
	f.blockParts = true
	st := newTestStorage(t, f)
	t.Cleanup(func() { close(f.unblock) })

	// Stream more than one part so the multipart upload starts, then cancel
	// only after the fake server has received the first UploadPart. The
	// adapter must still abort the started multipart upload: its cleanup
	// context is detached from the canceled upload context.
	//
	// [Ja] 複数パート分をストリーミングして multipart upload を開始させ、fake
	// サーバーが最初の UploadPart を受信した後にだけキャンセルする。アダプタは
	// 開始済みの multipart upload をそれでも中断しなければならない (cleanup
	// context はキャンセル済みの upload context から切り離されている)。
	pr, pw := io.Pipe()
	go func() {
		_, err := pw.Write(deterministicBytes(6 << 20))
		_ = pw.CloseWithError(err)
	}()
	// Unblock the writer goroutine in case the uploader stops reading early.
	//
	// [Ja] アップローダーが途中で読むのをやめた場合に備え、writer goroutine の
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
		t.Fatal("Upload() error = nil, want context cancellation error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Upload() error = %v, want errors.Is(err, context.Canceled)", err)
	}

	f.mu.Lock()
	created := f.multipartCreated
	aborted := f.aborted
	completed := f.completed
	f.mu.Unlock()

	if !created {
		t.Error("CreateMultipartUpload was not called")
	}
	if !aborted {
		t.Error("AbortMultipartUpload was not called after context cancellation")
	}
	if completed {
		t.Error("CompleteMultipartUpload was called for a canceled upload")
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
		t.Fatalf("Download() error = %v", err)
	}
	defer func() { _ = body.Close() }()

	if size != int64(len(content)) {
		t.Errorf("size = %d, want %d", size, len(content))
	}
	got, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("body length = %d, want %d (content mismatch)", len(got), len(content))
	}
	if err := body.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
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
		t.Fatalf("Download() error = %v", err)
	}

	// A client disconnect closes the stream mid-download; the returned body
	// must release the underlying HTTP response without hanging.
	//
	// [Ja] クライアント切断ではダウンロード途中でストリームを閉じる。
	// 返された body はハングせずに背後の HTTP レスポンスを解放しなければ
	// ならない。
	buf := make([]byte, 1024)
	if _, err := io.ReadFull(body, buf); err != nil {
		t.Fatalf("reading first chunk: %v", err)
	}
	if err := body.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if _, err := body.Read(buf); err == nil {
		t.Error("Read() after Close() error = nil, want error")
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
		t.Fatalf("Delete() error = %v", err)
	}

	f.mu.Lock()
	_, exists := f.objects[key]
	_, otherExists := f.objects[otherKey]
	deleted := append([]string(nil), f.deleted...)
	f.mu.Unlock()

	if exists {
		t.Error("object still exists after Delete()")
	}
	if !otherExists {
		t.Error("Delete() removed an unrelated object")
	}
	if len(deleted) != 1 || deleted[0] != key {
		t.Errorf("deleted keys = %v, want [%s]", deleted, key)
	}
}

func TestS3ExportStorage_Delete_MissingKeyIsSuccess(t *testing.T) {
	t.Parallel()

	f := newFakeS3()
	st := newTestStorage(t, f)

	// The fake answers 404 NoSuchKey for a missing key; the adapter must
	// treat it as success so retried cleanup jobs stay idempotent.
	//
	// [Ja] fake は存在しないキーに 404 NoSuchKey を返す。リトライされる
	// cleanup ジョブが冪等であるよう、アダプタはこれを成功として扱わなければ
	// ならない。
	if err := st.Delete(context.Background(), "exports/profile-id/missing.zip"); err != nil {
		t.Fatalf("Delete() error = %v, want nil for a missing key", err)
	}
}

func TestS3ExportStorage_Delete_NoSuchBucketIsError(t *testing.T) {
	t.Parallel()

	f := newFakeS3()
	f.deleteErrorCode = "NoSuchBucket"
	st := newTestStorage(t, f)

	err := st.Delete(context.Background(), "exports/profile-id/export-id.zip")
	if err == nil {
		t.Fatal("Delete() error = nil, want NoSuchBucket error")
	}
	if !strings.Contains(err.Error(), "NoSuchBucket") {
		t.Errorf("Delete() error = %v, want NoSuchBucket", err)
	}
}

func TestS3ExportStorage_ListPrefix(t *testing.T) {
	t.Parallel()

	f := newFakeS3()
	// Force pagination: 5 matching objects with 2 keys per page require 3
	// ListObjectsV2 calls.
	//
	// [Ja] ページングを強制する: 一致する 5 オブジェクトを 1 ページ 2 キーで
	// 返すと、ListObjectsV2 は 3 回呼ばれる。
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
	// An object of another feature sharing the bucket must not be listed.
	//
	// [Ja] バケットを共有する他機能のオブジェクトは一覧されてはならない。
	f.objects["images/profile-a/avatar.png"] = []byte("png-bytes")
	st := newTestStorage(t, f)

	gotModifiedAt := make(map[string]time.Time, len(exportKeys))
	err := st.ListPrefix(context.Background(), "exports/", "", func(key string, lastModified time.Time) error {
		gotModifiedAt[key] = lastModified
		return nil
	})
	if err != nil {
		t.Fatalf("ListPrefix() error = %v", err)
	}

	if !maps.Equal(gotModifiedAt, wantModifiedAt) {
		t.Errorf("ListPrefix() = %v, want %v", gotModifiedAt, wantModifiedAt)
	}

	f.mu.Lock()
	listCalls := f.listCalls
	f.mu.Unlock()
	if listCalls != 3 {
		t.Errorf("ListObjectsV2 calls = %d, want 3 (pagination not followed)", listCalls)
	}
}

// TestS3ExportStorage_ListPrefix_StartAfter pins the resume position. A caller
// whose walk is bounded hands the key it stopped at to the next walk, so the
// listing has to begin strictly after that key and the pagination that follows
// has to keep going forward rather than back to it.
//
// [Ja] TestS3ExportStorage_ListPrefix_StartAfter は再開位置を固定する。走査が有界な
// 呼び出し側は止まったキーを次の走査へ渡すため、一覧はそのキーより厳密に後ろから始まり、
// 続くページングはそこへ戻らず前進し続ける必要がある。
func TestS3ExportStorage_ListPrefix_StartAfter(t *testing.T) {
	t.Parallel()

	f := newFakeS3()
	// Two keys per page, so the walk resumes and then pages at least once.
	//
	// [Ja] 1 ページ 2 キーにして、再開したあと少なくとも 1 回はページングさせる。
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
		t.Fatalf("ListPrefix() error = %v", err)
	}

	want := []string{
		"exports/profile-b/export-3.zip",
		"exports/profile-c/export-4.zip",
		"exports/profile-c/export-5.zip",
	}
	if !slices.Equal(got, want) {
		t.Errorf("ListPrefix() = %v, want %v", got, want)
	}

	// Only the first request carries the resume position. The later ones resume
	// from their continuation token, which already stands for a position further
	// on, so sending the original key again could only pull the walk backwards.
	//
	// [Ja] 再開位置を運ぶのは最初のリクエストだけである。以降は continuation token の
	// 位置から再開し、token はすでに先の位置を表しているため、元のキーを再送しても走査を
	// 後ろへ引き戻すことしかできない。
	f.mu.Lock()
	gotStartAfters := slices.Clone(f.listStartAfters)
	f.mu.Unlock()

	wantStartAfters := []string{"exports/profile-a/export-2.zip", ""}
	if !slices.Equal(gotStartAfters, wantStartAfters) {
		t.Errorf("各 ListObjectsV2 の start-after = %v, want %v", gotStartAfters, wantStartAfters)
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
		t.Fatalf("ListPrefix() error = %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("ListPrefix() = %v, want empty", keys)
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
		t.Errorf("ListPrefix() error = %v, want errors.Is(err, wantErr)", err)
	}
}

func TestS3ExportStorage_ListPrefix_TruncatedWithoutTokenIsError(t *testing.T) {
	t.Parallel()

	f := newFakeS3()
	// One key per page with two matching objects makes the first page
	// truncated, and omitting the token leaves the adapter no way to advance.
	// It must fail instead of refetching the first page forever.
	//
	// [Ja] 1 ページ 1 キーで一致オブジェクトが 2 件だと最初のページが切り詰め
	// られ、token を省くとアダプタは次へ進めない。1 ページ目を永久に取り直さず
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
		t.Fatal("ListPrefix() error = nil, want error for a truncated page without a continuation token")
	}

	f.mu.Lock()
	listCalls := f.listCalls
	f.mu.Unlock()
	if listCalls != 1 {
		t.Errorf("ListObjectsV2 calls = %d, want 1 (adapter looped on the first page)", listCalls)
	}
}

func TestS3ExportStorage_ListPrefix_MissingLastModifiedIsError(t *testing.T) {
	t.Parallel()

	f := newFakeS3()
	// A Contents entry without LastModified gives the caller no time to judge
	// the grace period against, so the adapter must reject it rather than
	// yield a zero time.
	//
	// [Ja] LastModified の無い Contents は猶予期間を判定する時刻を呼び出し側に
	// 渡せないため、アダプタはゼロ値の時刻を yield せず拒否しなければならない。
	f.listOmitLastModified = true
	f.objects["exports/profile-a/export-1.zip"] = []byte("zip-bytes")
	st := newTestStorage(t, f)

	err := st.ListPrefix(context.Background(), "exports/", "", func(string, time.Time) error {
		return nil
	})
	if err == nil {
		t.Fatal("ListPrefix() error = nil, want error for a Contents entry without LastModified")
	}
}

// deterministicBytes returns size bytes with a deterministic pattern so that
// content comparisons detect reordering or truncation.
//
// [Ja] deterministicBytes は決定的なパターンの size バイトを返す。内容比較で
// 並び替えや欠落を検出できるようにする。
func deterministicBytes(size int) []byte {
	b := make([]byte, size)
	for i := range b {
		b[i] = byte(i % 251)
	}
	return b
}
