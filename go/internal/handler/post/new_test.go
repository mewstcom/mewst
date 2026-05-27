package post_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	handler "github.com/mewstcom/mewst/go/internal/handler/post"
	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

func TestNew(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewTestConfig(t)
	h := handler.NewHandler(cfg)

	// Set the CSRF token and locale on the context. /new sits under RequireAuth,
	// so the CSRF token is supplied via the context in production.
	//
	// [Ja] CSRF トークンとロケールをコンテキストに設定する。/new は RequireAuth
	// 配下のため、本番では CSRF トークンは context 経由で渡る。
	ctx := i18n.SetLocale(context.Background(), "ja")
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	req := httptest.NewRequest(http.MethodGet, "/new", nil)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.New(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusOK)
	}

	if contentType := rr.Header().Get("Content-Type"); !strings.Contains(contentType, "text/html") {
		t.Errorf("Content-Type が不正: got %v, want text/html", contentType)
	}

	body := rr.Body.String()
	checks := []string{
		"csrf_token",                   // CSRF トークン
		"disableSubmitButtons",         // 二重送信防止 (他フォームと共通)
		`name="content"`,               // 本文 textarea
		`action="/posts"`,              // フォーム送信先 (Rails の post_list_path)
		"いまなにしてる？",                     // 本文ラベル (post_new_content_label)
		"投稿する",                         // 送信ボタン (post_new_submit)
		`href="/home"`,                 // 認証後共通レイアウトの navbar
		"[&_.content]:fill-foreground", // navbar の new 項目がアクティブ表示
	}
	for _, want := range checks {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに %q が含まれていません", want)
		}
	}
}

func TestNew_Locales(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewTestConfig(t)
	h := handler.NewHandler(cfg)

	tests := []struct {
		name   string
		locale string
		label  string
		submit string
	}{
		{name: "Japanese", locale: "ja", label: "いまなにしてる？", submit: "投稿する"},
		// The apostrophe in the English label "What's happening?" is escaped by
		// templ, so assert a stable substring instead of the escaped form.
		// [Ja] 英語ラベル "What's happening?" のアポストロフィは templ がエスケープ
		// するため、エスケープ表現に依存せず部分文字列で検証する。
		{name: "English", locale: "en", label: "happening?", submit: "Post"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := i18n.SetLocale(context.Background(), tt.locale)
			ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

			req := httptest.NewRequest(http.MethodGet, "/new", nil)
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			h.New(rr, req)

			body := rr.Body.String()
			if !strings.Contains(body, tt.label) {
				t.Errorf("本文ラベル %q がレスポンスに含まれていません", tt.label)
			}
			if !strings.Contains(body, tt.submit) {
				t.Errorf("送信ボタン %q がレスポンスに含まれていません", tt.submit)
			}
		})
	}
}
