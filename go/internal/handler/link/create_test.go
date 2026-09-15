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

	"github.com/google/uuid"

	handler "github.com/mewstcom/mewst/go/internal/handler/link"
	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/ratelimit"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/validator"
)

// newLinkHandlerはテスト用トランザクション上で動くlinkリポジトリを持つ
// link.Handlerを構築する。本テストのhttptestサーバーはloopback (127.0.0.1) で
// リッスンするため、FetchLinkMetadataUsecaseのテストと同様にprivate host
// ブロックは無効化する (ブロック自体はそちらで検証済み)。
func newLinkHandler(t *testing.T, tx *sql.Tx) *handler.Handler {
	t.Helper()

	uc := usecase.NewFetchLinkMetadataUsecase(
		validator.NewLinkDataFetcherValidator(),
		repository.NewLinkRepository(testutil.QueriesWithTx(tx)),
		&http.Client{},
		false,
	)
	rateLimiter := ratelimit.NewLimiter(repository.NewRateLimitRepository(testutil.QueriesWithTx(tx)))
	return handler.NewHandler(uc, rateLimiter)
}

// postLinkRequestは指定の対象URLを持つPOST /linksリクエストを構築する。
// CSRFトークンとリクエスト主体のプロフィールは、本番でCSRF / RequireAuth
// ミドルウェアが行うのと同じ形でcontextに注入する。
func postLinkRequest(t *testing.T, profile *model.Profile, targetURL string) *http.Request {
	t.Helper()

	form := url.Values{}
	form.Set("target_url", targetURL)
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/links", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := i18n.SetLocale(context.Background(), "ja")
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = middleware.SetProfileToContext(ctx, profile)
	return req.WithContext(ctx)
}

func TestCreate_Success(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h := newLinkHandler(t, tx)

	// ページはcanonical linkを持たないため、canonical URLは対象URLに
	// フォールバックする (サーバーURLをレスポンスに埋め込む手間を避ける)。
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `<html><head>
			<meta property="og:title" content="OGP Title">
			<meta property="og:image" content="https://example.com/og.png">
			<title>Page Title</title>
		</head><body></body></html>`)
	}))
	defer server.Close()
	targetURL := server.URL + "/articles/1"

	profile := &model.Profile{ID: model.ProfileID(uuid.New())}
	req := postLinkRequest(t, profile, targetURL)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	checks := []string{
		`name="canonical_url"`,             // 投稿フォームに紐付くhidden input
		`value="` + targetURL + `"`,        // canonical_urlの値 (対象URLにフォールバック)
		"OGP Title",                        // リンクカードのタイトル
		"127.0.0.1",                        // リンクカードのドメイン
		`src="https://example.com/og.png"`, // OGP画像
		"リンクカードを削除",                        // 削除ボタンのaria-label (link_create_remove)
		"hx-on:click",                      // 削除ボタンが #link-formをクリアする
	}
	for _, want := range checks {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていません", want)
		}
	}
}

func TestCreate_ValidationError_EmptyTargetURL(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h := newLinkHandler(t, tx)

	profile := &model.Profile{ID: model.ProfileID(uuid.New())}
	req := postLinkRequest(t, profile, "")
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusUnprocessableEntity)
	}

	body := rr.Body.String()
	checks := []string{
		"入力してください",         // validation_required (対象URL必須エラー)
		`hx-post="/links"`, // プロンプトのフラグメントが再描画されている
		`name="target_url"`,
	}
	for _, want := range checks {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていません", want)
		}
	}
}

func TestCreate_ValidationError_FetchFailed(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h := newLinkHandler(t, tx)

	// 空のボディは取得失敗として扱われる (空の取得結果がフォームに取得エラーを
	// 積むRailsの挙動に対応)。
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	targetURL := server.URL + "/empty"

	profile := &model.Profile{ID: model.ProfileID(uuid.New())}
	req := postLinkRequest(t, profile, targetURL)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusUnprocessableEntity)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "URLの情報を取得できませんでした") {
		t.Error("レスポンスに取得失敗エラー (validation_link_fetch_failed) が含まれていません")
	}
	// 同じURLを再送信できるよう、対象URLがエコーバックされること。
	if !strings.Contains(body, `value="`+targetURL+`"`) {
		t.Error("対象URLがエコーバックされていません")
	}
}

func TestCreate_RateLimitExceeded(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h := newLinkHandler(t, tx)

	testutil.WaitForRateLimitWindow(t)

	// profile単位の上限 (10回/分) を使い切る。レートリミットは
	// バリデーションより先にチェックされるため、対象URLが空の不正リクエスト
	// でもカウントされ、外部サーバーへのアクセスなしで上限に到達できる。
	profile := &model.Profile{ID: model.ProfileID(uuid.New())}
	for i := 0; i < 10; i++ {
		rr := httptest.NewRecorder()
		h.Create(rr, postLinkRequest(t, profile, ""))
	}

	// 11回目のリクエストで上限を超過する。
	rr := httptest.NewRecorder()
	h.Create(rr, postLinkRequest(t, profile, ""))

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusUnprocessableEntity)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "リクエストが多すぎます") {
		t.Error("レート制限エラーメッセージ (validation_rate_limit_exceeded) が表示されていません")
	}
	// 時間を置いて再送信できるよう、プロンプトのフラグメントが再描画されること。
	if !strings.Contains(body, `hx-post="/links"`) {
		t.Error("プロンプトのフラグメントが再描画されていません")
	}

	// 別のprofileには影響しない: レート制限メッセージではなく通常の
	// バリデーションエラーが返ること (カウンターはprofile単位)。
	otherProfile := &model.Profile{ID: model.ProfileID(uuid.New())}
	otherRR := httptest.NewRecorder()
	h.Create(otherRR, postLinkRequest(t, otherProfile, ""))

	if otherRR.Code != http.StatusUnprocessableEntity {
		t.Fatalf("別profileのステータスコードが不正: 実測値 = %v、期待値 = %v", otherRR.Code, http.StatusUnprocessableEntity)
	}
	otherBody := otherRR.Body.String()
	if strings.Contains(otherBody, "リクエストが多すぎます") {
		t.Error("別profileがレート制限に巻き込まれています")
	}
	if !strings.Contains(otherBody, "入力してください") {
		t.Error("別profileに通常のバリデーションエラー (validation_required) が返っていません")
	}
}

func TestCreate_NoProfile_InternalServerError(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h := newLinkHandler(t, tx)

	form := url.Values{}
	form.Set("target_url", "https://example.com/articles/1")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/links", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := i18n.SetLocale(context.Background(), "ja")
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	// SetProfileToContextを呼ばず、profileをcontextに注入しない。
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusInternalServerError)
	}
}
