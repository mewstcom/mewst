package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBodyLimit_WithinLimit(t *testing.T) {
	t.Parallel()

	var receivedBody []byte
	handler := BodyLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("ボディの読み込みに失敗: %v", err)
			return
		}
		receivedBody = b
		w.WriteHeader(http.StatusOK)
	}))

	body := strings.Repeat("a", 1024) // 1 KiB
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコードが期待と異なる: got %d want %d", rr.Code, http.StatusOK)
	}
	if string(receivedBody) != body {
		t.Errorf("ボディが期待と異なる: got len=%d want len=%d", len(receivedBody), len(body))
	}
}

func TestBodyLimit_ExactLimit(t *testing.T) {
	t.Parallel()

	handler := BodyLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.ReadAll(r.Body); err != nil {
			t.Errorf("ボディの読み込みに失敗: %v", err)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	body := strings.Repeat("a", DefaultMaxBodyBytes)
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコードが期待と異なる: got %d want %d", rr.Code, http.StatusOK)
	}
}

func TestBodyLimit_OverLimitReturns413(t *testing.T) {
	t.Parallel()

	called := false
	handler := BodyLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	body := strings.Repeat("a", DefaultMaxBodyBytes+1)
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("ステータスコードが期待と異なる: got %d want %d", rr.Code, http.StatusRequestEntityTooLarge)
	}
	if !strings.Contains(rr.Body.String(), "Request Entity Too Large") {
		t.Errorf("レスポンスボディに期待する文字列が含まれない: body=%q", rr.Body.String())
	}
	if called {
		t.Error("上限超過時は下流ハンドラーを呼び出すべきでない")
	}
}

// TestBodyLimit_ContentLengthOverLimitRejectsEarly verifies that a request whose
// Content-Length exceeds the limit is rejected with 413 without reading the body.
// It checks that the downstream handler is never reached and that the request
// body is never read (verified via a counter).
//
// [Ja] TestBodyLimit_ContentLengthOverLimitRejectsEarly は Content-Length が上限を
// 超えている場合にボディを読み込まずに 413 を返すことを検証する。下流ハンドラーに
// 到達しないこと・リクエストボディがまったく読まれていないこと (カウンタで検証) を確認する。
func TestBodyLimit_ContentLengthOverLimitRejectsEarly(t *testing.T) {
	t.Parallel()

	called := false
	handler := BodyLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	// The actual body is short but Content-Length declares it over the limit.
	// If early rejection works, the body is never read and 413 is returned.
	//
	// [Ja] 実際のボディは短いが Content-Length を上限超過で申告する。早期拒否が効いていれば
	// ボディの中身は読まれずに 413 が返る。
	body := &readCounter{src: strings.NewReader("dummy")}
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	req.ContentLength = DefaultMaxBodyBytes + 1
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("ステータスコードが期待と異なる: got %d want %d", rr.Code, http.StatusRequestEntityTooLarge)
	}
	if !strings.Contains(rr.Body.String(), "Request Entity Too Large") {
		t.Errorf("レスポンスボディに期待する文字列が含まれない: body=%q", rr.Body.String())
	}
	if called {
		t.Error("Content-Length 超過時は下流ハンドラーを呼び出すべきでない")
	}
	if body.reads > 0 {
		t.Errorf("Content-Length による早期拒否時はボディを読むべきでない: reads=%d", body.reads)
	}
}

// readCounter is an io.Reader that counts how many times the body is read.
// Used to verify the body is never read when rejected early via Content-Length.
//
// [Ja] readCounter はボディの読み込み回数をカウントする io.Reader。
// Content-Length での早期拒否時にボディが読まれていないことを検証するために使う。
type readCounter struct {
	src   io.Reader
	reads int
}

func (r *readCounter) Read(p []byte) (int, error) {
	r.reads++
	return r.src.Read(p)
}

func TestBodyLimit_GetRequestPassesThrough(t *testing.T) {
	t.Parallel()

	called := false
	handler := BodyLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("ハンドラーが呼ばれていない")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコードが期待と異なる: got %d want %d", rr.Code, http.StatusOK)
	}
}

// TestBodyLimit_DownstreamCanReadFormValue verifies that r.ParseForm / r.FormValue
// keep working in downstream handlers after the body has been pre-read.
//
// [Ja] TestBodyLimit_DownstreamCanReadFormValue は先読み後に下流ハンドラーで
// r.ParseForm / r.FormValue が引き続き機能することを検証する。
func TestBodyLimit_DownstreamCanReadFormValue(t *testing.T) {
	t.Parallel()

	var receivedValue string
	handler := BodyLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedValue = r.FormValue("key")
		w.WriteHeader(http.StatusOK)
	}))

	body := "key=hello"
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコードが期待と異なる: got %d want %d", rr.Code, http.StatusOK)
	}
	if receivedValue != "hello" {
		t.Errorf("フォーム値が期待と異なる: got %q want %q", receivedValue, "hello")
	}
}
