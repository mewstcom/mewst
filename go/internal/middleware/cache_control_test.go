package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPrivateCache_SetsHeader(t *testing.T) {
	t.Parallel()

	handler := PrivateCache(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/settings/export", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコードが期待と異なる: got %d want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Cache-Control"); got != PrivateCacheControl {
		t.Errorf("Cache-Control = %q, want %q", got, PrivateCacheControl)
	}
}

// TestPrivateCache_SetsHeaderOnShortCircuit pins that a downstream middleware
// answering on its own still gets the header. The authentication middleware
// redirects an unauthenticated request without reaching the handler, and that
// redirect must not be cacheable by a shared cache either.
//
// [Ja] TestPrivateCache_SetsHeaderOnShortCircuit は、下流のミドルウェアが自分で
// 応答した場合にもヘッダーが付くことを固定する。認証ミドルウェアは未認証の
// リクエストをハンドラーへ渡さずリダイレクトするが、そのリダイレクトも共有
// キャッシュに保存されてはならない。
func TestPrivateCache_SetsHeaderOnShortCircuit(t *testing.T) {
	t.Parallel()

	handler := PrivateCache(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
	}))

	req := httptest.NewRequest(http.MethodGet, "/settings/export", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが期待と異なる: got %d want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Cache-Control"); got != PrivateCacheControl {
		t.Errorf("Cache-Control = %q, want %q", got, PrivateCacheControl)
	}
}
