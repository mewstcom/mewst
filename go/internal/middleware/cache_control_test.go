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
		t.Errorf("ステータスコードが期待と異なる: 実測値 = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Cache-Control"); got != PrivateCacheControl {
		t.Errorf("Cache-Control = %q、期待値 = %q", got, PrivateCacheControl)
	}
}

// TestPrivateCache_SetsHeaderOnShortCircuitは、下流のミドルウェアが自分で
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
		t.Errorf("ステータスコードが期待と異なる: 実測値 = %d、期待値 = %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Cache-Control"); got != PrivateCacheControl {
		t.Errorf("Cache-Control = %q、期待値 = %q", got, PrivateCacheControl)
	}
}
