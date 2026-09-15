package post_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
)

func TestNew(t *testing.T) {
	t.Parallel()

	h := newCreatePostHandler(t)

	// CSRFトークン・ロケール・現在プロフィールをコンテキストに設定する。/newは
	// RequireAuth配下のため、本番ではCSRFトークンとプロフィールはcontext経由で渡る。
	// プロフィールはnavbarのプロフィールリンク (/@{atname}) を駆動する。
	authorID := model.ProfileID(uuid.MustParse("11111111-1111-1111-1111-111111111111"))
	ctx := i18n.SetLocale(context.Background(), "ja")
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = middleware.SetProfileToContext(ctx, &model.Profile{ID: authorID, Atname: "alice"})

	req := httptest.NewRequest(http.MethodGet, "/new", nil)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.New(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	if contentType := rr.Header().Get("Content-Type"); !strings.Contains(contentType, "text/html") {
		t.Errorf("Content-Typeが不正: 実測値 = %v、期待値 = text/html", contentType)
	}

	body := rr.Body.String()
	checks := []string{
		"csrf_token",           // CSRFトークン
		"disableSubmitButtons", // 二重送信防止 (他フォームと共通)
		// 本文の下書き自動保存のPEフック (web/post_draft.ts)。投稿者のプロフィール
		// IDにスコープし、共有端末で前ユーザーの下書きが次のユーザーに漏れないようにする。
		`data-draft-key="post_draft:11111111-1111-1111-1111-111111111111"`,
		`name="content"`,       // 本文textarea
		`action="/posts"`,      // フォーム送信先 (Railsのpost_list_path)
		"いま何してる？",              // 本文ラベル (post_new_content_label)
		"投稿する",                 // 送信ボタン (post_new_submit)
		`<h1 class="sr-only">`, // 見出し階層を確立するsr-onlyのh1 (heading-hierarchy)
		"新規投稿",                 // h1の文言 (post_new_heading)
		`href="#main"`,         // レイアウトのスキップリンク (WCAG 2.4.1)
		"メインコンテンツへスキップ",        // スキップリンクのラベル
		`<main id="main"`,      // mainランドマーク (semantic-html)
		"M227.32,28.68",        // 送信ボタン先頭のpaper-plane-tiltアイコン (このアイコン固有のpath片)
		// フォーム上部に描画される戻る導線 (BackLinkコンポーネント):
		// referrerが同一オリジンのときback-linkスクリプトがhistory.back() に格上げする
		// /homeフォールバックリンク (data-back-linkがPEフック)。<button> ではなくmutedな
		// リンクとしてスタイルした <a> で、JS無しでもhrefで遷移し、フォームを誤送信しない。
		"data-back-link",
		`href="/home"`,
		"link-bare-muted-foreground",
		"戻る",
		// フォーム拡張: 文字数カウンター・autosize・リンクカード連携の配線
		`id="link-form"`,                       // リンクカードフラグメントのhtmxターゲット
		`data-link-card-path="/links/new"`,     // URL検出モジュールの取得先パス
		"data-autosize",                        // autosizeモジュールの対象マーカー
		`data-character-counter-for="content"`, // 文字数カウンターの対象textarea
		`data-character-counter-max="160"`,     // 文字数上限 (model.MaximumPostContentLength)
		`data-focus-textarea="content"`,        // 下部領域のフォーカスプロキシ: クリックで #contentにフォーカス
		"cursor-text",                          // 下部領域にtextareaと同じテキストカーソルを表示
		"data-focus-proxy-ignore",              // #link-formは除外しリンクカードのクリック・カーソルを維持
	}
	for _, want := range checks {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていません", want)
		}
	}

	// 唯一の戻るリンクは投稿フォームより前にあり、フォーム上部かつ操作行の外に
	// 置かれる必要がある。操作行は投稿ボタンだけを含み、justify-endで右端へ置く。
	backLinkIdx := strings.Index(body, "data-back-link")
	formIdx := strings.Index(body, `<form action="/posts"`)
	if backLinkIdx < 0 || formIdx < 0 || backLinkIdx > formIdx {
		t.Errorf("戻るリンクが投稿フォームより前にありません: backLinkIdx=%d, formIdx=%d", backLinkIdx, formIdx)
	}
	if got := strings.Count(body, "data-back-link"); got != 1 {
		t.Errorf("戻るリンク数 = %d、期待値 = 1", got)
	}

	const actionRowStart = `<div class="flex items-center justify-end"><button class="btn`
	if !strings.Contains(body, actionRowStart) {
		t.Errorf("投稿ボタンだけを右寄せする操作行%qがありません", actionRowStart)
	}
	const unexpectedActionRowStart = `<div class="flex items-center justify-between"><button class="btn`
	if strings.Contains(body, unexpectedActionRowStart) {
		t.Errorf("投稿ボタンをjustify-betweenで配置する操作行%qが含まれています", unexpectedActionRowStart)
	}

	// 初回表示のフォームにはcanonical_urlのhiddenフィールドを含めない
	// (リンクカードのエコーバック時のみ描画される)。
	if strings.Contains(body, `name="canonical_url"`) {
		t.Error("初回表示のフォームにcanonical_urlのhidden inputが含まれています")
	}

	// /newは認証後のnavbar付きレイアウト (layouts.Centered) で描画し、トップnavbar
	// (PC) とボトムnavbar (モバイル) がそれぞれ5項目メニューを描画する。navbar専用リンク
	// (検索) と、注入したプロフィールのatnameから生成されるプロフィールリンクを検証し、
	// ハンドラーが /newをnavbar付き中央寄せレイアウトへ配線していることを確認する。
	navbarChecks := []string{
		`href="/search"`,
		`href="/@alice"`,
		"sticky",
		"lg:hidden",
	}
	for _, want := range navbarChecks {
		if !strings.Contains(body, want) {
			t.Errorf("navbarを表示する /newに%qが含まれていません", want)
		}
	}

	// /newではnew項目がアクティブ。両navbarがメニューを描画するため、アクティブの
	// 塗りつぶしアイコンのfill上書きはちょうど2回 (メニューごとに1回) 現れる。
	if got := strings.Count(body, "[&_.content]:fill-foreground"); got != 2 {
		t.Errorf("アクティブ表示のfillクラス数 = %d、期待値 = 2 (newがトップ / ボトムnavbarでアクティブ)", got)
	}
}

func TestNew_WithoutProfile(t *testing.T) {
	t.Parallel()

	h := newCreatePostHandler(t)

	ctx := i18n.SetLocale(context.Background(), "ja")
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	req := httptest.NewRequest(http.MethodGet, "/new", nil)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.New(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	// プロフィールが無いときは下書きキーをスコープするユーザー別IDが無いため、
	// 自動保存のPEフック自体を描画しない。web/post_draft.tsは
	// textarea[data-draft-key] しか拾わないため、属性を省くことが「共有端末で次の
	// 訪問者も読めるキーに下書きを保存しない」ことの担保になる。
	if body := rr.Body.String(); strings.Contains(body, "data-draft-key") {
		t.Error("プロフィール不在時のレスポンスにdata-draft-keyが含まれています")
	}
}

func TestNew_ContentPrefill(t *testing.T) {
	t.Parallel()

	h := newCreatePostHandler(t)

	// GETフォームは ?content= から本文textareaを事前入力し、送信時まで意図的に
	// 検証を遅延する。テンプレートは値を `>{ data.Content }</textarea>` として描画するため、
	// 各ケースは閉じタグ直前に来るはずの部分文字列を検証する。
	overLimit := strings.Repeat("a", 161)

	tests := []struct {
		name     string
		query    string
		wantBody string
	}{
		{name: "クエリパラメータからtextareaを事前入力する", query: "?content=hello", wantBody: "hello</textarea>"},
		{name: "パラメータが無いときはtextareaを空のままにする", query: "", wantBody: "></textarea>"},
		{name: "上限を超える事前入力もそのまま表示する (検証は送信時に行う)", query: "?content=" + overLimit, wantBody: overLimit + "</textarea>"},
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
				t.Errorf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusOK)
			}
			if body := rr.Body.String(); !strings.Contains(body, tt.wantBody) {
				t.Errorf("textareaに%qが含まれていません", tt.wantBody)
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
		{name: "日本語", locale: "ja", label: "いま何してる？", submit: "投稿する"},
		// 英語ラベル "What's happening?" のアポストロフィはtemplがエスケープ
		// するため、エスケープ表現に依存せず部分文字列で検証する。
		{name: "英語", locale: "en", label: "happening?", submit: "Post"},
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
				t.Errorf("本文ラベル%qがレスポンスに含まれていません", tt.label)
			}
			if !strings.Contains(body, tt.submit) {
				t.Errorf("送信ボタン%qがレスポンスに含まれていません", tt.submit)
			}
		})
	}
}
