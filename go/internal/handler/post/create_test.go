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

// noopJobInserter satisfies dispatcher.JobInserter without enqueuing anything,
// so the handler tests can exercise CreatePostUsecase without a running River
// client (the fanout enqueue happens after commit and must not fail the test).
//
// [Ja] noopJobInserter は何も enqueue せずに dispatcher.JobInserter を満たす。
// これにより、River クライアントを動かさずにハンドラーテストで CreatePostUsecase を
// 実行できる (fanout の enqueue はコミット後に走るが、テストを失敗させてはならない)。
type noopJobInserter struct{}

func (noopJobInserter) Insert(_ context.Context, _ river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	return &rivertype.JobInsertResult{}, nil
}

// newCreatePostHandler builds a post.Handler backed by a CreatePostUsecase whose
// repositories run against the shared test DB. CreatePostUsecase opens its own
// transaction, so callers commit prerequisite rows (profile, mewst-web) before
// invoking Create.
//
// [Ja] newCreatePostHandler は共有テスト DB に対して動く CreatePostUsecase を持つ
// post.Handler を構築する。CreatePostUsecase は独自のトランザクションを開くため、
// 呼び出し側は Create を呼ぶ前に前提行 (profile・mewst-web) をコミットしておく。
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

// postRequest builds a POST /posts request carrying the given form content, with
// the CSRF token and current profile injected into the context the way the CSRF
// and RequireAuth middleware do in production.
//
// [Ja] postRequest は指定の本文を持つ POST /posts リクエストを構築する。CSRF
// トークンと現在プロフィールは、本番で CSRF / RequireAuth ミドルウェアが行うのと
// 同じ形で context に注入する。
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

	// This commits an oauth_applications row with uid = mewst-web (UNIQUE
	// index), so serialize against the other tests that commit or assert the
	// absence of the same row (e.g. the CreatePostUsecase tests, which run as a
	// separate process on the shared DB).
	// [Ja] uid = mewst-web (UNIQUE インデックス) の oauth_applications 行を
	// コミットするため、同じ行をコミットする / 不在を前提とする他テスト
	// (共有 DB 上で別プロセスとして実行される CreatePostUsecase のテスト等) と
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
		t.Fatalf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}
	if location := rr.Header().Get("Location"); location != "/home" {
		t.Errorf("リダイレクト先が不正: got %v, want /home", location)
	}

	// 成功フラッシュメッセージの Cookie が設定されていること
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

	// 投稿が DB に作成されていること
	var count int
	if err := db.QueryRow(
		`SELECT count(*) FROM posts WHERE profile_id = $1 AND content = $2`,
		uuid.UUID(authorID), "Hello, Mewst!",
	).Scan(&count); err != nil {
		t.Fatalf("posts の取得に失敗: %v", err)
	}
	if count != 1 {
		t.Errorf("作成された投稿件数 = %d, want 1", count)
	}
}

func TestCreate_NormalizesNewlines(t *testing.T) {
	t.Parallel()

	// Same mewst-web row serialization rationale as TestCreate_Success.
	// [Ja] mewst-web 行の直列化理由は TestCreate_Success と同じ。
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
	// Submit a body with CRLF newlines, mirroring what a browser form sends.
	// [Ja] ブラウザのフォーム送信を模して CRLF の改行を含む本文を送信する。
	req := postRequest(t, &model.Profile{ID: authorID}, "line1\r\nline2\r\nline3")
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}

	// The stored body must carry LF newlines (no CR), so a newline counts as a
	// single code point everywhere.
	// [Ja] 保存本文は LF 改行 (CR を含まない) でなければならず、改行が全箇所で
	// 1 コードポイントになる。
	var stored string
	if err := db.QueryRow(
		`SELECT content FROM posts WHERE profile_id = $1`, uuid.UUID(authorID),
	).Scan(&stored); err != nil {
		t.Fatalf("posts の取得に失敗: %v", err)
	}
	if want := "line1\nline2\nline3"; stored != want {
		t.Errorf("保存された本文が正規化されていません: got %q, want %q", stored, want)
	}
}

func TestCreate_ValidationError_EmptyContent(t *testing.T) {
	t.Parallel()

	// An empty body fails validation before any DB access, so no prerequisite
	// rows are needed; a synthetic profile ID suffices to build the input.
	// [Ja] 空の本文は DB アクセス前にバリデーションで弾かれるため、前提行は不要。
	// 入力構築には合成のプロフィール ID で足りる。
	h := newCreatePostHandler(t)
	profile := &model.Profile{ID: model.ProfileID(uuid.New())}
	req := postRequest(t, profile, "   ") // 空白のみ → presence エラー
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	body := rr.Body.String()
	checks := []string{
		"入力してください",        // validation_required (本文必須エラー)
		`action="/posts"`, // フォームが再描画されている
		`name="content"`,  // 本文 textarea
	}
	for _, want := range checks {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに %q が含まれていません", want)
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

	// 161 runes exceed the 160-character limit and fail validation before DB access.
	// [Ja] 161 文字は 160 文字上限を超え、DB アクセス前にバリデーションで弾かれる。
	h := newCreatePostHandler(t)
	profile := &model.Profile{ID: model.ProfileID(uuid.New())}
	req := postRequest(t, profile, strings.Repeat("あ", 161))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusUnprocessableEntity)
	}
	if body := rr.Body.String(); !strings.Contains(body, "160文字以内で入力してください") {
		t.Error("レスポンスに文字数超過エラーが含まれていません")
	}
}

func TestCreate_ValidationError_PreservesCanonicalURL(t *testing.T) {
	t.Parallel()

	// An empty body fails validation before reaching the persistence path. The
	// submitted canonical_url must survive the 422 re-render so the attached
	// link card is not lost when the post body is invalid. The URL here matches
	// no links row, so this also covers the fallback: a bare hidden input
	// instead of a re-rendered link card.
	// [Ja] 空の本文は永続化パスに到達する前にバリデーションで弾かれる。送信した
	// canonical_url は 422 再描画でも保持され、本文が不正でも紐付けたリンクカードが
	// 失われないこと。この URL は links 行に一致しないため、リンクカードの再描画
	// ではなく hidden input のみ残すフォールバックも併せて検証する。
	h := newCreatePostHandler(t)
	profile := &model.Profile{ID: model.ProfileID(uuid.New())}

	form := url.Values{}
	form.Set("content", "") // presence エラー
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
		t.Fatalf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusUnprocessableEntity)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `name="canonical_url"`) {
		t.Error("再描画フォームに canonical_url の hidden input が含まれていません")
	}
	if !strings.Contains(body, `value="https://example.com/article"`) {
		t.Error("canonical_url の値がエコーバックされていません")
	}

	// The echoed hidden input must sit inside the #link-form container so the
	// URL detection module sees the container as occupied and does not attach a
	// second link (4-3).
	// [Ja] エコーバックされた hidden input は #link-form コンテナの内側に置かれ、
	// URL 検出モジュールがコンテナを使用中とみなして 2 つ目のリンクを紐付けない
	// ことを保証する (4-3)。
	linkFormIdx := strings.Index(body, `id="link-form"`)
	canonicalIdx := strings.Index(body, `name="canonical_url"`)
	counterIdx := strings.Index(body, `data-character-counter-for`)
	if linkFormIdx == -1 || counterIdx == -1 {
		t.Fatal("再描画フォームに #link-form または文字数カウンターが含まれていません")
	}
	if canonicalIdx < linkFormIdx || counterIdx < canonicalIdx {
		t.Error("canonical_url の hidden input が #link-form コンテナの内側にありません")
	}

	// The unknown URL resolves to no link, so no link card (and no remove
	// button) is re-rendered.
	// [Ja] 未知の URL はリンクに解決されないため、リンクカード (と削除ボタン) は
	// 再描画されないこと。
	if strings.Contains(body, "リンクカードを削除") {
		t.Error("未知の canonical_url なのにリンクカードが再描画されています")
	}
}

func TestCreate_ValidationError_RendersAttachedLinkCard(t *testing.T) {
	t.Parallel()

	// When the echoed canonical_url matches an existing link, the 422 re-render
	// shows the full link card (preview + remove button + hidden canonical_url)
	// inside #link-form, so the attachment stays visible and removable instead
	// of surviving only as an invisible hidden input.
	// [Ja] エコーバックされた canonical_url が既存リンクに一致する場合、422 再描画は
	// #link-form 内にリンクカード一式 (プレビュー + 削除ボタン + hidden の
	// canonical_url) を表示する。紐付けが不可視の hidden input としてだけ残るのでは
	// なく、カードとして見え削除ボタンで外せること。
	db := testutil.GetTestDB()

	canonicalURL := "https://example.com/attached-card-" + uuid.NewString()

	// The handler's link lookup runs outside the test transaction, so commit
	// the prerequisite link row and clean it up afterwards.
	// [Ja] ハンドラーのリンク取得はテストのトランザクション外で走るため、前提の
	// link 行はコミットし、終了時に削除する。
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
	form.Set("content", "") // presence error. [Ja] presence エラー
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
		t.Fatalf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusUnprocessableEntity)
	}
	body := rr.Body.String()
	checks := []string{
		"Attached Card Test", // link card preview (link title). [Ja] リンクカードのプレビュー (リンクのタイトル)
		"リンクカードを削除",          // remove button aria-label (link_create_remove). [Ja] 削除ボタンの aria-label
		`name="canonical_url"`,
		`value="` + canonicalURL + `"`,
	}
	for _, want := range checks {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに %q が含まれていません", want)
		}
	}

	// The re-rendered card (and its hidden input) must sit inside the
	// #link-form container so the URL detection sees it as occupied.
	// [Ja] 再描画されたカード (とその hidden input) は #link-form コンテナの
	// 内側に置かれ、URL 検出がコンテナを使用中とみなすこと。
	linkFormIdx := strings.Index(body, `id="link-form"`)
	cardIdx := strings.Index(body, "Attached Card Test")
	counterIdx := strings.Index(body, `data-character-counter-for`)
	if linkFormIdx == -1 || counterIdx == -1 {
		t.Fatal("再描画フォームに #link-form または文字数カウンターが含まれていません")
	}
	if cardIdx < linkFormIdx || counterIdx < cardIdx {
		t.Error("リンクカードが #link-form コンテナの内側にありません")
	}
}

func TestCreate_NoProfile_InternalServerError(t *testing.T) {
	t.Parallel()

	// The profile is supplied by RequireAuth in production; its absence is a
	// defensive case (the middleware did not run as expected). Build the request
	// without injecting a profile and confirm Create returns 500 before any DB
	// access. Validation runs first, so a valid body is needed to reach the
	// profile check.
	//
	// [Ja] 本番では profile は RequireAuth が供給する。その不在は防御的なケース
	// (ミドルウェアが想定どおり動いていない) なので、profile を注入せずにリクエストを
	// 構築し、Create が DB アクセス前に 500 を返すことを確認する。バリデーションが
	// 先に走るため、profile チェックに到達させるには有効な本文が必要。
	h := newCreatePostHandler(t)

	form := url.Values{}
	form.Set("content", "Hello, Mewst!")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/posts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := i18n.SetLocale(context.Background(), "ja")
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	// Do not call SetProfileToContext, leaving the profile absent from the context.
	// [Ja] SetProfileToContext を呼ばず、profile を context に注入しない。
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusInternalServerError)
	}
}
