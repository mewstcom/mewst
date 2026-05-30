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

	return handler.NewHandler(cfg, flashMgr, createPostUC)
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
	// Not parallel: this commits an oauth_applications row with uid = mewst-web,
	// which has a UNIQUE index, so it must not run alongside other subtests that
	// commit the same uid.
	// [Ja] 並列化しない: uid = mewst-web (UNIQUE インデックス) の oauth_applications 行を
	// コミットするため、同じ uid をコミットする他テストと同時実行してはならない。

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

	// An empty body fails validation before any DB access. The submitted
	// canonical_url must survive the 422 re-render so the attached link card is
	// not lost when the post body is invalid.
	// [Ja] 空の本文は DB アクセス前にバリデーションで弾かれる。送信した canonical_url は
	// 422 再描画でも保持され、本文が不正でも紐付けたリンクカードが失われないこと。
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
