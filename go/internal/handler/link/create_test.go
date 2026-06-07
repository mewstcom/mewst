package link_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	handler "github.com/mewstcom/mewst/go/internal/handler/link"
	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/validator"
)

// newLinkHandler builds a link.Handler whose link repository runs on the test
// transaction. The httptest servers used by these tests listen on loopback
// (127.0.0.1), so private host blocking is disabled, mirroring the
// FetchLinkMetadataUsecase tests (the block itself is verified there).
//
// [Ja] newLinkHandler はテスト用トランザクション上で動く link リポジトリを持つ
// link.Handler を構築する。本テストの httptest サーバーは loopback (127.0.0.1) で
// リッスンするため、FetchLinkMetadataUsecase のテストと同様に private host
// ブロックは無効化する (ブロック自体はそちらで検証済み)。
func newLinkHandler(t *testing.T, tx *sql.Tx) *handler.Handler {
	t.Helper()

	uc := usecase.NewFetchLinkMetadataUsecase(
		validator.NewLinkDataFetcherValidator(),
		repository.NewLinkRepository(testutil.QueriesWithTx(tx)),
		&http.Client{},
		false,
	)
	return handler.NewHandler(uc)
}

// postLinkRequest builds a POST /links request carrying the given target URL,
// with the CSRF token injected into the context the way the CSRF middleware
// does in production.
//
// [Ja] postLinkRequest は指定の対象 URL を持つ POST /links リクエストを構築する。
// CSRF トークンは、本番で CSRF ミドルウェアが行うのと同じ形で context に注入する。
func postLinkRequest(t *testing.T, targetURL string) *http.Request {
	t.Helper()

	form := url.Values{}
	form.Set("target_url", targetURL)
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/links", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := i18n.SetLocale(context.Background(), "ja")
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	return req.WithContext(ctx)
}

func TestCreate_Success(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h := newLinkHandler(t, tx)

	// The page has no canonical link, so the canonical URL falls back to the
	// target URL (no closure dance to embed the server URL into the response).
	// [Ja] ページは canonical link を持たないため、canonical URL は対象 URL に
	// フォールバックする (サーバー URL をレスポンスに埋め込む手間を避ける)。
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `<html><head>
			<meta property="og:title" content="OGP Title">
			<meta property="og:image" content="https://example.com/og.png">
			<title>Page Title</title>
		</head><body></body></html>`)
	}))
	defer server.Close()
	targetURL := server.URL + "/articles/1"

	req := postLinkRequest(t, targetURL)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	checks := []string{
		`name="canonical_url"`,             // 投稿フォームに紐付く hidden input
		`value="` + targetURL + `"`,        // canonical_url の値 (対象 URL にフォールバック)
		"OGP Title",                        // リンクカードのタイトル
		"127.0.0.1",                        // リンクカードのドメイン
		`src="https://example.com/og.png"`, // OGP 画像
		"リンクカードを削除",                        // 削除ボタンの aria-label (link_create_remove)
		"hx-on:click",                      // 削除ボタンが #link-form をクリアする
	}
	for _, want := range checks {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに %q が含まれていません", want)
		}
	}
}

func TestCreate_ValidationError_EmptyTargetURL(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h := newLinkHandler(t, tx)

	req := postLinkRequest(t, "")
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	body := rr.Body.String()
	checks := []string{
		"入力してください",         // validation_required (対象 URL 必須エラー)
		`hx-post="/links"`, // プロンプトのフラグメントが再描画されている
		`name="target_url"`,
	}
	for _, want := range checks {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに %q が含まれていません", want)
		}
	}
}

func TestCreate_ValidationError_FetchFailed(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h := newLinkHandler(t, tx)

	// An empty body is treated as a fetch failure (mirrors the Rails behavior
	// where a blank fetch result adds a fetch error to the form).
	// [Ja] 空のボディは取得失敗として扱われる (空の取得結果がフォームに取得エラーを
	// 積む Rails の挙動に対応)。
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	targetURL := server.URL + "/empty"

	req := postLinkRequest(t, targetURL)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "URLの情報を取得できませんでした") {
		t.Error("レスポンスに取得失敗エラー (validation_link_fetch_failed) が含まれていません")
	}
	// The target URL is echoed back so the user can retry the same URL.
	// [Ja] 同じ URL を再送信できるよう、対象 URL がエコーバックされること。
	if !strings.Contains(body, `value="`+targetURL+`"`) {
		t.Error("対象 URL がエコーバックされていません")
	}
}
