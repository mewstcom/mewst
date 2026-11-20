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

	// フラグメントは投稿フォームからhtmxで取得されるため、CSRFトークンは
	// 本番でCSRFミドルウェアが行うのと同じ形でcontext経由で渡す。
	ctx := i18n.SetLocale(context.Background(), "ja")
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	targetURL := "https://example.com/articles/awesome-post"
	req := httptest.NewRequest(http.MethodGet, "/links/new?url="+url.QueryEscape(targetURL), nil)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.New(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	checks := []string{
		`hx-post="/links"`,          // プロンプトのボタンがPOST /linksに送信する
		`hx-target="#link-form"`,    // 投稿フォーム内の #link-formコンテナにスワップする
		"csrf_token",                // POST /links用のCSRFトークン
		`name="target_url"`,         // 検出したURLをhiddenで運ぶ
		`value="` + targetURL + `"`, // 対象URLがそのまま埋め込まれる
		// ボタンラベルはhost + pathを25 runeに短縮して表示する
		// (link_new_submit + ShortenHostAndPath)。
		"リンクカードを追加する: example.com/articles/a...",
	}
	for _, want := range checks {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていません", want)
		}
	}
}

func TestNew_WithoutURL(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h := newLinkHandler(t, tx)

	// "url" パラメータが無い / パース不能でもプロンプト自体は描画される
	// (Railsのnewビューも同様でhost_and_pathが空になるだけ)。URLの検証は
	// POST /links時のみ行う。
	ctx := i18n.SetLocale(context.Background(), "ja")
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	req := httptest.NewRequest(http.MethodGet, "/links/new", nil)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.New(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), "リンクカードを追加する") {
		t.Error("レスポンスにプロンプトのボタンラベルが含まれていません")
	}
}
