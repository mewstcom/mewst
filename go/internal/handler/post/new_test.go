package post_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
)

func TestNew(t *testing.T) {
	t.Parallel()

	h := newCreatePostHandler(t)

	// Set the CSRF token, locale, and current profile on the context. /new sits
	// under RequireAuth, so in production the CSRF token and profile are supplied via
	// the context; the profile drives the navbar's profile link (/@{atname}).
	//
	// [Ja] CSRF トークン・ロケール・現在プロフィールをコンテキストに設定する。/new は
	// RequireAuth 配下のため、本番では CSRF トークンとプロフィールは context 経由で渡る。
	// プロフィールは navbar のプロフィールリンク (/@{atname}) を駆動する。
	ctx := i18n.SetLocale(context.Background(), "ja")
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = middleware.SetProfileToContext(ctx, &model.Profile{Atname: "alice"})

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
		"data-form-guard",      // leave guard PE hook (web/form_guard.ts). [Ja] 離脱ガードの PE フック (web/form_guard.ts)
		`name="content"`,       // 本文 textarea
		`action="/posts"`,      // フォーム送信先 (Rails の post_list_path)
		"いま何してる？",              // 本文ラベル (post_new_content_label)
		"投稿する",                 // 送信ボタン (post_new_submit)
		`<h1 class="sr-only">`, // sr-only h1 that establishes the heading hierarchy. [Ja] 見出し階層を確立する sr-only の h1 (heading-hierarchy)
		"新規投稿",                 // h1 text (post_new_heading). [Ja] h1 の文言 (post_new_heading)
		`href="#main"`,         // layout skip link (WCAG 2.4.1). [Ja] レイアウトのスキップリンク (WCAG 2.4.1)
		"メインコンテンツへスキップ",        // skip link label. [Ja] スキップリンクのラベル
		`<main id="main"`,      // main landmark (semantic-html). [Ja] main ランドマーク (semantic-html)
		"M227.32,28.68",        // submit button leading paper-plane-tilt icon (path unique to that icon). [Ja] 送信ボタン先頭の paper-plane-tilt アイコン (このアイコン固有の path 片)
		// Cancel affordance rendered inside the form's action row (CancelButton
		// component): a /home fallback link upgraded to history.back() by the back-link
		// script when the referrer is same-origin (data-back-link is the PE hook). It is
		// an <a> styled as an outline button, not a <button>, so it navigates via href
		// without JS and cannot accidentally submit the form.
		//
		// [Ja] フォームの操作行の中に描画されるキャンセル導線 (CancelButton コンポーネント):
		// referrer が同一オリジンのとき back-link スクリプトが history.back() に格上げする
		// /home フォールバックリンク (data-back-link が PE フック)。<button> ではなく輪郭のみ
		// (outline) のボタン風にスタイルした <a> で、JS 無しでも href で遷移し、フォームを誤送信しない。
		"data-back-link", // キャンセルリンクの JS フック (PE)
		`href="/home"`,   // キャンセル導線のフォールバック先
		"btn-outline",    // outline (輪郭のみ) ボタンのスタイル
		"キャンセル",          // キャンセルボタンのラベル (cancel)
		// Form enhancements: wiring for the character counter, autosize, and the
		// link card integration.
		// [Ja] フォーム拡張: 文字数カウンター・autosize・リンクカード連携の配線
		`id="link-form"`,                       // link card fragment htmx target. [Ja] リンクカードフラグメントの htmx ターゲット
		`data-link-card-path="/links/new"`,     // URL detection fetch path. [Ja] URL 検出モジュールの取得先パス
		"data-autosize",                        // autosize module marker. [Ja] autosize モジュールの対象マーカー
		`data-character-counter-for="content"`, // character counter target textarea. [Ja] 文字数カウンターの対象 textarea
		`data-character-counter-max="160"`,     // content length limit (model.MaximumPostContentLength). [Ja] 文字数上限
		`data-focus-textarea="content"`,        // lower-area focus proxy: clicking it focuses #content. [Ja] 下部領域のフォーカスプロキシ: クリックで #content にフォーカス
		"cursor-text",                          // lower area shows the textarea's text cursor. [Ja] 下部領域に textarea と同じテキストカーソルを表示
		"data-focus-proxy-ignore",              // #link-form opts out so the link card's clicks/cursor survive. [Ja] #link-form は除外しリンクカードのクリック・カーソルを維持
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

	// /new renders on the authenticated navbar layout (layouts.Centered), so the
	// top navbar (desktop) and bottom navbar (mobile) each render the five-item menu.
	// Assert a navbar-only link (search) and the profile link built from the injected
	// profile's atname to confirm both navbars render. This verifies that the handler
	// wires /new to the navbar-bearing centered layout.
	//
	// [Ja] /new は認証後の navbar 付きレイアウト (layouts.Centered) で描画し、トップ navbar
	// (PC) とボトム navbar (モバイル) がそれぞれ 5 項目メニューを描画する。navbar 専用リンク
	// (検索) と、注入したプロフィールの atname から生成されるプロフィールリンクを検証し、
	// ハンドラーが /new を navbar 付き中央寄せレイアウトへ配線していることを確認する。
	navbarChecks := []string{
		`href="/search"`,
		`href="/@alice"`,
		"sticky",
		"lg:hidden",
	}
	for _, want := range navbarChecks {
		if !strings.Contains(body, want) {
			t.Errorf("navbar を表示する /new に %q が含まれていません", want)
		}
	}

	// The new item is the active one on /new. Both navbars render the menu, so the
	// active filled-icon fill override appears exactly twice (once per menu).
	//
	// [Ja] /new では new 項目がアクティブ。両 navbar がメニューを描画するため、アクティブの
	// 塗りつぶしアイコンの fill 上書きはちょうど 2 回 (メニューごとに 1 回) 現れる。
	if got := strings.Count(body, "[&_.content]:fill-foreground"); got != 2 {
		t.Errorf("アクティブ表示の fill クラス数 = %d, want 2 (new がトップ / ボトム navbar でアクティブ)", got)
	}
}

func TestNew_ContentPrefill(t *testing.T) {
	t.Parallel()

	h := newCreatePostHandler(t)

	// The GET form pre-fills the content textarea from ?content= and intentionally
	// defers validation until submit. The template renders the value as
	// `>{ data.Content }</textarea>`, so each case asserts the substring that must sit
	// just before the closing tag.
	//
	// [Ja] GET フォームは ?content= から本文 textarea を事前入力し、送信時まで意図的に
	// 検証を遅延する。テンプレートは値を `>{ data.Content }</textarea>` として描画するため、
	// 各ケースは閉じタグ直前に来るはずの部分文字列を検証する。
	overLimit := strings.Repeat("a", 161)

	tests := []struct {
		name     string
		query    string
		wantBody string
	}{
		{name: "prefills the textarea from the query parameter", query: "?content=hello", wantBody: "hello</textarea>"},
		{name: "leaves the textarea empty without the parameter", query: "", wantBody: "></textarea>"},
		{name: "shows an over-limit prefill as-is (validation deferred to submit)", query: "?content=" + overLimit, wantBody: overLimit + "</textarea>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := i18n.SetLocale(context.Background(), "ja")
			ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

			req := httptest.NewRequest(http.MethodGet, "/new"+tt.query, nil)
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			h.New(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusOK)
			}
			if body := rr.Body.String(); !strings.Contains(body, tt.wantBody) {
				t.Errorf("textarea に %q が含まれていません", tt.wantBody)
			}
		})
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
		{name: "Japanese", locale: "ja", label: "いま何してる？", submit: "投稿する"},
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
