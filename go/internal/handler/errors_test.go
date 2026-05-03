package handler_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mewstcom/mewst/go/internal/handler"
	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/templates/pages/errors"
)

// serveWithI18n は i18n.Middleware を経由したリクエストを実行してレスポンスを返す
func serveWithI18n(t *testing.T, h http.HandlerFunc, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	rr := httptest.NewRecorder()
	i18n.Middleware(h).ServeHTTP(rr, req)
	return rr
}

func TestNotFound(t *testing.T) {
	t.Parallel()

	t.Run("ステータスコード404を返す", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
		rr := serveWithI18n(t, handler.NotFound, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusNotFound)
		}
	})

	t.Run("日本語のエラーメッセージを含む", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
		req.Header.Set("Accept-Language", "ja")
		rr := serveWithI18n(t, handler.NotFound, req)

		body := rr.Body.String()

		checks := []string{
			"お探しのページは見つかりませんでした",
			"ホームに戻る",
			"🫥",
			`lang="ja"`,
		}

		for _, expected := range checks {
			if !strings.Contains(body, expected) {
				t.Errorf("レスポンスに %q が含まれていません", expected)
			}
		}
	})

	t.Run("英語のエラーメッセージを含む", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
		req.Header.Set("Accept-Language", "en")
		rr := serveWithI18n(t, handler.NotFound, req)

		body := rr.Body.String()

		checks := []string{
			"The page you are looking for could not be found",
			"Back to home",
			"🫥",
			`lang="en"`,
		}

		for _, expected := range checks {
			if !strings.Contains(body, expected) {
				t.Errorf("レスポンスに %q が含まれていません", expected)
			}
		}
	})

	t.Run("ホームへのリンクが含まれる", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
		rr := serveWithI18n(t, handler.NotFound, req)

		body := rr.Body.String()
		if !strings.Contains(body, `href="/"`) {
			t.Error("ホームへのリンクが含まれていません")
		}
	})
}

func TestBadGateway(t *testing.T) {
	t.Parallel()

	t.Run("ステータスコード502を返す", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/some-path", nil)
		rr := serveWithI18n(t, handler.BadGateway, req)

		if rr.Code != http.StatusBadGateway {
			t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusBadGateway)
		}
	})

	t.Run("日本語のエラーメッセージを含む", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/some-path", nil)
		req.Header.Set("Accept-Language", "ja")
		rr := serveWithI18n(t, handler.BadGateway, req)

		body := rr.Body.String()

		checks := []string{
			"申し訳ございません。現在サービスに接続できません。",
			"しばらくしてから再度お試しください。",
			"ホームに戻る",
			"⚠️",
			`lang="ja"`,
		}

		for _, expected := range checks {
			if !strings.Contains(body, expected) {
				t.Errorf("レスポンスに %q が含まれていません", expected)
			}
		}
	})

	t.Run("英語のエラーメッセージを含む", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/some-path", nil)
		req.Header.Set("Accept-Language", "en")
		rr := serveWithI18n(t, handler.BadGateway, req)

		body := rr.Body.String()

		checks := []string{
			"Sorry, we are unable to connect to the service.",
			"Please try again later.",
			"Back to home",
			"⚠️",
			`lang="en"`,
		}

		for _, expected := range checks {
			if !strings.Contains(body, expected) {
				t.Errorf("レスポンスに %q が含まれていません", expected)
			}
		}
	})

	t.Run("ホームへのリンクが含まれる", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/some-path", nil)
		rr := serveWithI18n(t, handler.BadGateway, req)

		body := rr.Body.String()
		if !strings.Contains(body, `href="/"`) {
			t.Error("ホームへのリンクが含まれていません")
		}
	})
}

func TestBadGatewayTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		locale   string
		expected []string
	}{
		{
			name:   "日本語でレンダリングされる",
			locale: "ja",
			expected: []string{
				"申し訳ございません。現在サービスに接続できません。",
				"しばらくしてから再度お試しください。",
				"ホームに戻る",
				"⚠️",
				`lang="ja"`,
				`href="/"`,
			},
		},
		{
			name:   "英語でレンダリングされる",
			locale: "en",
			expected: []string{
				"Sorry, we are unable to connect to the service.",
				"Please try again later.",
				"Back to home",
				"⚠️",
				`lang="en"`,
				`href="/"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := i18n.SetLocale(t.Context(), tt.locale)

			var buf bytes.Buffer
			err := errors.BadGateway().Render(ctx, &buf)
			if err != nil {
				t.Fatalf("テンプレートのレンダリングに失敗: %v", err)
			}

			body := buf.String()
			for _, expected := range tt.expected {
				if !strings.Contains(body, expected) {
					t.Errorf("レスポンスに %q が含まれていません", expected)
				}
			}
		})
	}
}

func TestNotFoundTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		locale   string
		expected []string
	}{
		{
			name:   "日本語でレンダリングされる",
			locale: "ja",
			expected: []string{
				"お探しのページは見つかりませんでした",
				"ホームに戻る",
				"🫥",
				`lang="ja"`,
				`href="/"`,
			},
		},
		{
			name:   "英語でレンダリングされる",
			locale: "en",
			expected: []string{
				"The page you are looking for could not be found",
				"Back to home",
				"🫥",
				`lang="en"`,
				`href="/"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := i18n.SetLocale(t.Context(), tt.locale)

			var buf bytes.Buffer
			err := errors.NotFound().Render(ctx, &buf)
			if err != nil {
				t.Fatalf("テンプレートのレンダリングに失敗: %v", err)
			}

			body := buf.String()
			for _, expected := range tt.expected {
				if !strings.Contains(body, expected) {
					t.Errorf("レスポンスに %q が含まれていません", expected)
				}
			}
		})
	}
}
