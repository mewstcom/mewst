package httperror_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mewstcom/mewst/go/internal/httperror"
)

func TestNotFound(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	rr := httptest.NewRecorder()
	httperror.NotFound(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusNotFound)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type が不正: got %v, want %q", ct, "text/html; charset=utf-8")
	}
}

func TestBadGateway(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/some-path", nil)
	rr := httptest.NewRecorder()
	httperror.BadGateway(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusBadGateway)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type が不正: got %v, want %q", ct, "text/html; charset=utf-8")
	}
}
