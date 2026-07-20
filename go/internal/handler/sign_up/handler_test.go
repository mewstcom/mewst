package sign_up_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/dispatcher"
	handler "github.com/mewstcom/mewst/go/internal/handler/sign_up"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/ratelimit"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/validator"
)

// mockTurnstile はテスト用のTurnstile検証モック
type mockTurnstile struct {
	shouldSucceed bool
}

func (m *mockTurnstile) Verify(_ context.Context, _ string) (bool, error) {
	return m.shouldSucceed, nil
}

// mockInserter はテスト用のモック inserter
type mockInserter struct{}

func (m *mockInserter) Insert(_ context.Context, _ river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	return &rivertype.JobInsertResult{}, nil
}

// setupTestHandler はテスト用のハンドラーとテストデータをセットアップする
func setupTestHandler(t *testing.T, tx *sql.Tx, turnstileSuccess bool) (*handler.Handler, *config.Config) {
	t.Helper()

	cfg := testutil.NewTestConfig(t)

	// トランザクションを使用するリポジトリを作成
	userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
	actorRepo := repository.NewActorRepository(testutil.QueriesWithTx(tx))
	sessionRepo := repository.NewSessionRepository(testutil.QueriesWithTx(tx))
	emailConfirmRepo := repository.NewEmailConfirmationRepository(testutil.QueriesWithTx(tx))
	rateLimitRepo := repository.NewRateLimitRepository(testutil.QueriesWithTx(tx))

	sessionMgr := session.NewManager(sessionRepo, actorRepo, userRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)
	inserter := &mockInserter{}
	d := dispatcher.NewDispatcher(inserter)
	turnstile := &mockTurnstile{shouldSucceed: turnstileSuccess}
	rateLimiter := ratelimit.NewLimiter(rateLimitRepo)

	signUpValidator := validator.NewSignUpCreateValidator(userRepo)
	createSignUpUC := usecase.NewCreateSignUpUsecase(signUpValidator, emailConfirmRepo, d)
	h := handler.NewHandler(cfg, sessionMgr, flashMgr, createSignUpUC, turnstile, rateLimiter)

	return h, cfg
}

func TestNew(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx, true)

	// CSRFトークンをコンテキストに設定
	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	req := httptest.NewRequest(http.MethodGet, "/sign_up", nil)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.New(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusOK)
	}

	// レスポンスにサインアップフォームが含まれているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "csrf_token") {
		t.Error("CSRFトークンがフォームに含まれていません")
	}
	if !strings.Contains(body, "email") {
		t.Error("メールアドレスフィールドがフォームに含まれていません")
	}
	if !strings.Contains(body, `aria-label="Mewst"`) {
		t.Error("ロゴリンクにアクセシブルネームが含まれていません")
	}
}

func TestNew_ContainsSignInLink(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx, true)

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	req := httptest.NewRequest(http.MethodGet, "/sign_up", nil)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.New(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusOK)
	}

	// レスポンスにログインリンクが含まれているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "/sign_in") {
		t.Error("ログインリンクがフォームに含まれていません")
	}
}

func TestCreate_Success(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx, true)

	// CSRFトークンをコンテキストに設定
	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	// フォームデータを作成
	form := url.Values{}
	form.Set("email", "newuser@example.com")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/sign_up", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	// リダイレクトを検証
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}

	// リダイレクト先を検証
	location := rr.Header().Get("Location")
	if location != "/email_confirmation" {
		t.Errorf("リダイレクト先が不正: got %v, want /email_confirmation", location)
	}

	// email_confirmation_idクッキーが設定されているか確認
	cookies := rr.Result().Cookies()
	var emailConfirmCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == session.EmailConfirmationCookieName {
			emailConfirmCookie = c
			break
		}
	}
	if emailConfirmCookie == nil {
		t.Error("email_confirmation_idクッキーが設定されていません")
	}

	// フラッシュメッセージクッキーが設定されているか確認
	var flashCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == session.FlashCookieName {
			flashCookie = c
			break
		}
	}
	if flashCookie == nil {
		t.Error("フラッシュメッセージクッキーが設定されていません")
	}
}

func TestCreate_EmptyEmail(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx, true)

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	// 空のメールアドレスで送信試行
	form := url.Values{}
	form.Set("email", "")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/sign_up", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// バリデーションエラーが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "入力してください") {
		t.Error("必須フィールドのバリデーションエラーメッセージが表示されていません")
	}
}

func TestCreate_InvalidEmail(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx, true)

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	// 不正なメールアドレスで送信試行
	form := url.Values{}
	form.Set("email", "invalid-email")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/sign_up", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	// ステータスコードを検証 (422 Unprocessable Entity)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// エラーメッセージが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "正しいメールアドレス") {
		t.Error("バリデーションエラーメッセージが表示されていません")
	}
}

func TestCreate_TurnstileFailed(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx, false) // Turnstile検証失敗

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	form := url.Values{}
	form.Set("email", "newuser@example.com")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "invalid-token")

	req := httptest.NewRequest(http.MethodPost, "/sign_up", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// Turnstileエラーメッセージが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "ロボット検証に失敗") {
		t.Error("Turnstileエラーメッセージが表示されていません")
	}
}

func TestCreate_RateLimitExceeded(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx, true)

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	// レート制限を超過するためにリクエストを5回送信 (制限: 5回/分)
	for i := 0; i < 5; i++ {
		form := url.Values{}
		form.Set("email", "ratelimit@example.com")
		form.Set("csrf_token", "test-csrf-token")
		form.Set("cf-turnstile-response", "test-token")

		req := httptest.NewRequest(http.MethodPost, "/sign_up", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()

		h.Create(rr, req)
	}

	// 6回目のリクエストでレート制限超過
	form := url.Values{}
	form.Set("email", "ratelimit@example.com")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/sign_up", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// レート制限エラーメッセージが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "リクエストが多すぎます") {
		t.Error("レート制限エラーメッセージが表示されていません")
	}
}

func TestNew_WithBackParameter(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx, true)

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	req := httptest.NewRequest(http.MethodGet, "/sign_up?back=/settings", nil)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.New(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, `name="back"`) {
		t.Error("back hiddenフィールドがフォームに含まれていません")
	}
	if !strings.Contains(body, `value="/settings"`) {
		t.Error("back hiddenフィールドの値が正しくありません")
	}
}

func TestCreate_SuccessWithBackParameter(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx, true)

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	form := url.Values{}
	form.Set("email", "back-success@example.com")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")
	form.Set("back", "/settings")

	req := httptest.NewRequest(http.MethodPost, "/sign_up", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}

	// /email_confirmation?back=%2Fsettings へリダイレクトすることを検証
	location := rr.Header().Get("Location")
	wantLocation := "/email_confirmation?back=%2Fsettings"
	if location != wantLocation {
		t.Errorf("リダイレクト先が不正: got %v, want %v", location, wantLocation)
	}
}

func TestCreate_SuccessWithUnsafeBackParameter(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx, true)

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	form := url.Values{}
	form.Set("email", "back-unsafe@example.com")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")
	form.Set("back", "https://evil.com")

	req := httptest.NewRequest(http.MethodPost, "/sign_up", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}

	// 危険な back URL は破棄して /email_confirmation へ素のリダイレクト
	location := rr.Header().Get("Location")
	if location != "/email_confirmation" {
		t.Errorf("リダイレクト先が不正: got %v, want /email_confirmation", location)
	}
}

func TestCreate_EmailAlreadyTaken(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx, true)

	// 既存のユーザーを作成
	testutil.NewUserBuilder(t, tx).
		WithEmail("existing@example.com").
		WithPasswordDigest("$2a$10$test").
		Build()

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	// 既存のメールアドレスで送信試行
	form := url.Values{}
	form.Set("email", "existing@example.com")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/sign_up", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// エラーメッセージが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "既に登録されています") {
		t.Error("メールアドレス重複エラーメッセージが表示されていません")
	}
}
