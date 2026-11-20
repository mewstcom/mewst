package post_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	handler "github.com/mewstcom/mewst/go/internal/handler/post"
	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/validator"
)

// noopJobInserterは何もenqueueせずにdispatcher.JobInserterを満たす。
// これにより、Riverクライアントを動かさずにハンドラーテストでCreatePostUsecaseを
// 実行できる (fanoutのenqueueはコミット後に走るが、テストを失敗させてはならない)。
type noopJobInserter struct{}

func (noopJobInserter) Insert(_ context.Context, _ river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	return &rivertype.JobInsertResult{}, nil
}

// newCreatePostHandlerは共有テストDBに対して動くCreatePostUsecaseを持つ
// post.Handlerを構築する。CreatePostUsecaseは独自のトランザクションを開くため、
// 呼び出し側はCreateを呼ぶ前に前提行 (profile・mewst-web) をコミットしておく。
func newCreatePostHandler(t *testing.T) *handler.Handler {
	t.Helper()

	db := testutil.GetTestDB()
	cfg := testutil.NewTestConfig(t)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	q := query.New(db)
	createPostUC := usecase.NewCreatePostUsecase(
		db,
		validator.NewPostCreateValidator(),
		repository.NewOauthApplicationRepository(q),
		repository.NewLinkRepository(q),
		repository.NewPostRepository(q),
		repository.NewPostLinkRepository(q),
		repository.NewProfileRepository(q),
		repository.NewHomeTimelinePostRepository(q),
		dispatcher.NewDispatcher(noopJobInserter{}),
	)
	getLinkUC := usecase.NewGetLinkUsecase(repository.NewLinkRepository(q))

	return handler.NewHandler(cfg, flashMgr, createPostUC, getLinkUC)
}

// postRequestは指定の本文を持つPOST /postsリクエストを構築する。CSRF
// トークンと現在プロフィールは、本番でCSRF / RequireAuthミドルウェアが行うのと
// 同じ形でcontextに注入する。
func postRequest(t *testing.T, profile *model.Profile, content string) *http.Request {
	t.Helper()

	form := url.Values{}
	form.Set("content", content)
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/posts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := i18n.SetLocale(context.Background(), "ja")
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = middleware.SetProfileToContext(ctx, profile)
	return req.WithContext(ctx)
}

func TestCreate_Success(t *testing.T) {
	t.Parallel()

	// uid = mewst-web (UNIQUEインデックス) のoauth_applications行を
	// コミットするため、同じ行をコミットする / 不在を前提とする他テスト
	// (共有DB上で別プロセスとして実行されるCreatePostUsecaseのテスト等) と
	// 直列化する。
	testutil.AcquireMewstWebLock(t)

	db := testutil.GetTestDB()

	setupTx, err := db.Begin()
	if err != nil {
		t.Fatalf("セットアップ用トランザクションの開始に失敗: %v", err)
	}
	defer func() { _ = setupTx.Rollback() }()
	authorID := testutil.NewProfileBuilder(t, setupTx).Build()
	testutil.NewOauthApplicationBuilder(t, setupTx).WithUID(model.MewstWebUID).Build()
	if err := setupTx.Commit(); err != nil {
		t.Fatalf("前提データのコミットに失敗: %v", err)
	}
	t.Cleanup(func() {
		id := uuid.UUID(authorID)
		_, _ = db.Exec(`DELETE FROM home_timeline_posts WHERE post_id IN (SELECT id FROM posts WHERE profile_id = $1)`, id)
		_, _ = db.Exec(`DELETE FROM posts WHERE profile_id = $1`, id)
		_, _ = db.Exec(`DELETE FROM profiles WHERE id = $1`, id)
		_, _ = db.Exec(`DELETE FROM oauth_applications WHERE uid = $1`, model.MewstWebUID)
	})

	h := newCreatePostHandler(t)
	req := postRequest(t, &model.Profile{ID: authorID}, "Hello, Mewst!")
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusFound)
	}
	if location := rr.Header().Get("Location"); location != "/home" {
		t.Errorf("リダイレクト先が不正: 実測値 = %v、期待値 = /home", location)
	}

	// 成功フラッシュメッセージのCookieが設定されていること
	var flashCookie *http.Cookie
	for _, c := range rr.Result().Cookies() {
		if c.Name == session.FlashCookieName {
			flashCookie = c
			break
		}
	}
	if flashCookie == nil {
		t.Error("フラッシュメッセージクッキーが設定されていません")
	}

	// 投稿がDBに作成されていること
	var count int
	if err := db.QueryRow(
		`SELECT count(*) FROM posts WHERE profile_id = $1 AND content = $2`,
		uuid.UUID(authorID), "Hello, Mewst!",
	).Scan(&count); err != nil {
		t.Fatalf("postsの取得に失敗: %v", err)
	}
	if count != 1 {
		t.Errorf("作成された投稿件数 = %d、期待値 = 1", count)
	}
}

func TestCreate_NormalizesNewlines(t *testing.T) {
	t.Parallel()

	// mewst-web行の直列化理由はTestCreate_Successと同じ。
	testutil.AcquireMewstWebLock(t)

	db := testutil.GetTestDB()

	setupTx, err := db.Begin()
	if err != nil {
		t.Fatalf("セットアップ用トランザクションの開始に失敗: %v", err)
	}
	defer func() { _ = setupTx.Rollback() }()
	authorID := testutil.NewProfileBuilder(t, setupTx).Build()
	testutil.NewOauthApplicationBuilder(t, setupTx).WithUID(model.MewstWebUID).Build()
	if err := setupTx.Commit(); err != nil {
		t.Fatalf("前提データのコミットに失敗: %v", err)
	}
	t.Cleanup(func() {
		id := uuid.UUID(authorID)
		_, _ = db.Exec(`DELETE FROM home_timeline_posts WHERE post_id IN (SELECT id FROM posts WHERE profile_id = $1)`, id)
		_, _ = db.Exec(`DELETE FROM posts WHERE profile_id = $1`, id)
		_, _ = db.Exec(`DELETE FROM profiles WHERE id = $1`, id)
		_, _ = db.Exec(`DELETE FROM oauth_applications WHERE uid = $1`, model.MewstWebUID)
	})

	h := newCreatePostHandler(t)
	// ブラウザのフォーム送信を模してCRLFの改行を含む本文を送信する。
	req := postRequest(t, &model.Profile{ID: authorID}, "line1\r\nline2\r\nline3")
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusFound)
	}

	// 保存本文はLF改行 (CRを含まない) でなければならず、改行が全箇所で
	// 1コードポイントになる。
	var stored string
	if err := db.QueryRow(
		`SELECT content FROM posts WHERE profile_id = $1`, uuid.UUID(authorID),
	).Scan(&stored); err != nil {
		t.Fatalf("postsの取得に失敗: %v", err)
	}
	if want := "line1\nline2\nline3"; stored != want {
		t.Errorf("保存された本文が正規化されていません: 実測値 = %q、期待値 = %q", stored, want)
	}
}

func TestCreate_ValidationError_EmptyContent(t *testing.T) {
	t.Parallel()

	// 空の本文はDBアクセス前にバリデーションで弾かれるため、前提行は不要。
	// 入力構築には合成のプロフィールIDで足りる。
	h := newCreatePostHandler(t)
	profile := &model.Profile{ID: model.ProfileID(uuid.New())}
	req := postRequest(t, profile, "   ") // 空白のみ → presenceエラー
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusUnprocessableEntity)
	}

	body := rr.Body.String()
	checks := []string{
		"入力してください",        // validation_required (本文必須エラー)
		`action="/posts"`, // フォームが再描画されている
		`name="content"`,  // 本文textarea
	}
	for _, want := range checks {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていません", want)
		}
	}

	// リダイレクトせず、投稿成功フラッシュも設定されないこと
	for _, c := range rr.Result().Cookies() {
		if c.Name == session.FlashCookieName {
			t.Error("バリデーションエラー時にフラッシュメッセージクッキーが設定されています")
		}
	}
}

func TestCreate_ValidationError_TooLongContent(t *testing.T) {
	t.Parallel()

	// 161文字は160文字上限を超え、DBアクセス前にバリデーションで弾かれる。
	h := newCreatePostHandler(t)
	profile := &model.Profile{ID: model.ProfileID(uuid.New())}
	req := postRequest(t, profile, strings.Repeat("あ", 161))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusUnprocessableEntity)
	}
	if body := rr.Body.String(); !strings.Contains(body, "160文字以内で入力してください") {
		t.Error("レスポンスに文字数超過エラーが含まれていません")
	}
}

func TestCreate_ValidationError_PreservesCanonicalURL(t *testing.T) {
	t.Parallel()

	// 空の本文は永続化パスに到達する前にバリデーションで弾かれる。送信した
	// canonical_urlは422再描画でも保持され、本文が不正でも紐付けたリンクカードが
	// 失われないこと。このURLはlinks行に一致しないため、リンクカードの再描画
	// ではなくhidden inputのみ残すフォールバックも併せて検証する。
	h := newCreatePostHandler(t)
	profile := &model.Profile{ID: model.ProfileID(uuid.New())}

	form := url.Values{}
	form.Set("content", "") // presenceエラー
	form.Set("canonical_url", "https://example.com/article")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/posts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx := i18n.SetLocale(context.Background(), "ja")
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = middleware.SetProfileToContext(ctx, profile)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusUnprocessableEntity)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `name="canonical_url"`) {
		t.Error("再描画フォームにcanonical_urlのhidden inputが含まれていません")
	}
	if !strings.Contains(body, `value="https://example.com/article"`) {
		t.Error("canonical_urlの値がエコーバックされていません")
	}

	// エコーバックされたhidden inputは #link-formコンテナの内側に置かれ、
	// URL検出モジュールがコンテナを使用中とみなして2つ目のリンクを紐付けない
	// ことを保証する (4-3)。
	linkFormIdx := strings.Index(body, `id="link-form"`)
	canonicalIdx := strings.Index(body, `name="canonical_url"`)
	counterIdx := strings.Index(body, `data-character-counter-for`)
	if linkFormIdx == -1 || counterIdx == -1 {
		t.Fatal("再描画フォームに #link-formまたは文字数カウンターが含まれていません")
	}
	if canonicalIdx < linkFormIdx || counterIdx < canonicalIdx {
		t.Error("canonical_urlのhidden inputが #link-formコンテナの内側にありません")
	}

	// 未知のURLはリンクに解決されないため、リンクカード (と削除ボタン) は
	// 再描画されないこと。
	if strings.Contains(body, "リンクカードを削除") {
		t.Error("未知のcanonical_urlなのにリンクカードが再描画されています")
	}
}

func TestCreate_ValidationError_RendersAttachedLinkCard(t *testing.T) {
	t.Parallel()

	// エコーバックされたcanonical_urlが既存リンクに一致する場合、422再描画は
	// #link-form内にリンクカード一式 (プレビュー + 削除ボタン + hiddenの
	// canonical_url) を表示する。紐付けが不可視のhidden inputとしてだけ残るのでは
	// なく、カードとして見え削除ボタンで外せること。
	db := testutil.GetTestDB()

	canonicalURL := "https://example.com/attached-card-" + uuid.NewString()

	// ハンドラーのリンク取得はテストのトランザクション外で走るため、前提の
	// link行はコミットし、終了時に削除する。
	setupTx, err := db.Begin()
	if err != nil {
		t.Fatalf("セットアップ用トランザクションの開始に失敗: %v", err)
	}
	defer func() { _ = setupTx.Rollback() }()
	testutil.NewLinkBuilder(t, setupTx).
		WithCanonicalURL(canonicalURL).
		WithTitle("Attached Card Test").
		Build()
	if err := setupTx.Commit(); err != nil {
		t.Fatalf("前提データのコミットに失敗: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM links WHERE canonical_url = $1`, canonicalURL)
	})

	h := newCreatePostHandler(t)
	profile := &model.Profile{ID: model.ProfileID(uuid.New())}

	form := url.Values{}
	form.Set("content", "") // presenceエラー
	form.Set("canonical_url", canonicalURL)
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/posts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx := i18n.SetLocale(context.Background(), "ja")
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = middleware.SetProfileToContext(ctx, profile)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusUnprocessableEntity)
	}
	body := rr.Body.String()
	checks := []string{
		"Attached Card Test", // リンクカードのプレビュー (リンクのタイトル)
		"リンクカードを削除",          // 削除ボタンのaria-label (link_create_remove)
		`name="canonical_url"`,
		`value="` + canonicalURL + `"`,
	}
	for _, want := range checks {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていません", want)
		}
	}

	// 再描画されたカード (とそのhidden input) は #link-formコンテナの
	// 内側に置かれ、URL検出がコンテナを使用中とみなすこと。
	linkFormIdx := strings.Index(body, `id="link-form"`)
	cardIdx := strings.Index(body, "Attached Card Test")
	counterIdx := strings.Index(body, `data-character-counter-for`)
	if linkFormIdx == -1 || counterIdx == -1 {
		t.Fatal("再描画フォームに #link-formまたは文字数カウンターが含まれていません")
	}
	if cardIdx < linkFormIdx || counterIdx < cardIdx {
		t.Error("リンクカードが #link-formコンテナの内側にありません")
	}
}

func TestCreate_NoProfile_InternalServerError(t *testing.T) {
	t.Parallel()

	// 本番ではprofileはRequireAuthが供給する。その不在は防御的なケース
	// (ミドルウェアが想定どおり動いていない) なので、profileを注入せずにリクエストを
	// 構築し、CreateがDBアクセス前に500を返すことを確認する。バリデーションが
	// 先に走るため、profileチェックに到達させるには有効な本文が必要。
	h := newCreatePostHandler(t)

	form := url.Values{}
	form.Set("content", "Hello, Mewst!")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/posts", strings.NewReader(form.Encode()))
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
