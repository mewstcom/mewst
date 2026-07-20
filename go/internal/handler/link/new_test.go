package link_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

func TestNew(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h := newLinkHandler(t, tx)

	// The fragment is fetched via htmx from the post form, so the CSRF token is
	// supplied via the context the way the CSRF middleware does in production.
	// [Ja] フラグメントは投稿フォームから htmx で取得されるため、CSRF トークンは
	// 本番で CSRF ミドルウェアが行うのと同じ形で context 経由で渡す。
	ctx := i18n.SetLocale(context.Background(), "ja")
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	targetURL := "https://example.com/articles/awesome-post"
	req := httptest.NewRequest(http.MethodGet, "/links/new?url="+url.QueryEscape(targetURL), nil)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.New(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	checks := []string{
		`hx-post="/links"`,          // プロンプトのボタンが POST /links に送信する
		`hx-target="#link-form"`,    // 投稿フォーム内の #link-form コンテナにスワップする
		"csrf_token",                // POST /links 用の CSRF トークン
		`name="target_url"`,         // 検出した URL を hidden で運ぶ
		`value="` + targetURL + `"`, // 対象 URL がそのまま埋め込まれる
		// The button label shows the host + path truncated to 25 runes
		// (link_new_submit + ShortenHostAndPath).
		// [Ja] ボタンラベルは host + path を 25 rune に短縮して表示する
		// (link_new_submit + ShortenHostAndPath)。
		"リンクカードを追加する: example.com/articles/a...",
	}
	for _, want := range checks {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに %q が含まれていません", want)
		}
	}
}

func TestNew_WithoutURL(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h := newLinkHandler(t, tx)

	// A missing or unparsable "url" parameter still renders the prompt (the
	// Rails new view behaves the same: host_and_path is just blank). The URL is
	// validated only on POST /links.
	// [Ja] "url" パラメータが無い / パース不能でもプロンプト自体は描画される
	// (Rails の new ビューも同様で host_and_path が空になるだけ)。URL の検証は
	// POST /links 時のみ行う。
	ctx := i18n.SetLocale(context.Background(), "ja")
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	req := httptest.NewRequest(http.MethodGet, "/links/new", nil)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.New(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), "リンクカードを追加する") {
		t.Error("レスポンスにプロンプトのボタンラベルが含まれていません")
	}
}
