package post_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/middleware"
)

func TestNew(t *testing.T) {
	t.Parallel()

	h := newCreatePostHandler(t)

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
		"csrf_token",           // CSRF トークン
		"disableSubmitButtons", // 二重送信防止 (他フォームと共通)
		`name="content"`,       // 本文 textarea
		`action="/posts"`,      // フォーム送信先 (Rails の post_list_path)
		"いまなにしてる？",             // 本文ラベル (post_new_content_label)
		"投稿する",                 // 送信ボタン (post_new_submit)
		// Compose layout back affordance: a /home fallback link upgraded to
		// history.back() by the back-link script when the referrer is same-origin.
		// [Ja] compose レイアウトの戻る導線: referrer が同一オリジンのとき back-link
		// スクリプトが history.back() に格上げする /home フォールバックリンク
		"data-back-link", // 戻るリンクの JS フック
		`href="/home"`,   // 戻る導線のフォールバック先
		"戻る",             // 戻るリンクのラベル (compose_back)
		// Form enhancements: wiring for the character counter, autosize, and the
		// link card integration.
		// [Ja] フォーム拡張: 文字数カウンター・autosize・リンクカード連携の配線
		`id="link-form"`,                       // link card fragment htmx target. [Ja] リンクカードフラグメントの htmx ターゲット
		`data-link-card-path="/links/new"`,     // URL detection fetch path. [Ja] URL 検出モジュールの取得先パス
		"data-autosize",                        // autosize module marker. [Ja] autosize モジュールの対象マーカー
		`data-character-counter-for="content"`, // character counter target textarea. [Ja] 文字数カウンターの対象 textarea
		`data-character-counter-max="160"`,     // content length limit (model.MaximumPostContentLength). [Ja] 文字数上限
	}
	for _, want := range checks {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに %q が含まれていません", want)
		}
	}

	// A fresh form must not carry an empty canonical_url hidden field; it is
	// rendered only when a link card is echoed back.
	// [Ja] 初回表示のフォームには canonical_url の hidden フィールドを含めない
	// (リンクカードのエコーバック時のみ描画される)。
	if strings.Contains(body, `name="canonical_url"`) {
		t.Error("初回表示のフォームに canonical_url の hidden input が含まれています")
	}

	// The compose layout omits the navbar, so navbar-only links (e.g. search) must
	// not appear. This locks /new to the focused compose layout instead of the
	// shared authenticated layout.
	// [Ja] compose レイアウトは navbar を持たないため、navbar 専用リンク (検索など) は
	// 出力されない。これにより /new が共通の認証後レイアウトではなく集中作成 compose
	// レイアウトに固定されることを保証する。
	if strings.Contains(body, `href="/search"`) {
		t.Error("compose レイアウトの /new に navbar の検索リンクが含まれています")
	}
}

func TestNew_Locales(t *testing.T) {
	t.Parallel()

	h := newCreatePostHandler(t)

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
