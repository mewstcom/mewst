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
	handler "github.com/mewstcom/mewst/go/internal/handler/sign_up"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/ratelimit"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/templates"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
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

func (m *mockInserter) Insert(_ context.Context, _ river.JobArgs) (*rivertype.JobInsertResult, error) {
	return &rivertype.JobInsertResult{}, nil
}

// setupTestHandler はテスト用のハンドラーとテストデータをセットアップする
func setupTestHandler(t *testing.T, db *sql.DB, tx *sql.Tx, turnstileSuccess bool) (*handler.Handler, *config.Config) {
	t.Helper()

	cfg := &config.Config{
		Env:              "test",
		Port:             "3000",
		Domain:           "localhost",
		CookieDomain:     "localhost",
		SessionSecure:    false,
		SessionHTTPOnly:  true,
		TurnstileSiteKey: "test-site-key",
	}

	// トランザクションを使用するリポジトリを作成
	userRepo := repository.NewUserRepository(db).WithTx(tx)
	actorRepo := repository.NewActorRepository(db).WithTx(tx)
	sessionRepo := repository.NewSessionRepository(db).WithTx(tx)
	emailConfirmRepo := repository.NewEmailConfirmationRepository(db).WithTx(tx)
	rateLimitRepo := repository.NewRateLimitRepository(db).WithTx(tx)

	sessionMgr := session.NewManager(sessionRepo, actorRepo, userRepo, cfg)
	inserter := &mockInserter{}
	createEmailConfirmUC := usecase.NewCreateEmailConfirmationUsecase(emailConfirmRepo, inserter)
	turnstile := &mockTurnstile{shouldSucceed: turnstileSuccess}
	rateLimiter := ratelimit.NewLimiter(rateLimitRepo)

	h := handler.NewHandler(cfg, sessionMgr, userRepo, createEmailConfirmUC, turnstile, rateLimiter)

	return h, cfg
}

func TestNew(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, cfg := setupTestHandler(t, db, tx, true)

	// CSRFトークンをコンテキストに設定
	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, cfg)

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
}

func TestNew_ContainsSignInLink(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, cfg := setupTestHandler(t, db, tx, true)

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, cfg)

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

	db, tx := testutil.SetupTestDB(t)
	h, cfg := setupTestHandler(t, db, tx, true)

	// CSRFトークンをコンテキストに設定
	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, cfg)

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

	db, tx := testutil.SetupTestDB(t)
	h, cfg := setupTestHandler(t, db, tx, true)

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, cfg)

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

	db, tx := testutil.SetupTestDB(t)
	h, cfg := setupTestHandler(t, db, tx, true)

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, cfg)

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

	// ステータスコードを検証（422 Unprocessable Entity）
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

	db, tx := testutil.SetupTestDB(t)
	h, cfg := setupTestHandler(t, db, tx, false) // Turnstile検証失敗

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, cfg)

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

func TestCreate_EmailAlreadyTaken(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, cfg := setupTestHandler(t, db, tx, true)

	// 既存のユーザーを作成
	testutil.NewUserBuilder(t, tx).
		WithEmail("existing@example.com").
		WithPasswordDigest("$2a$10$test").
		Build()

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, cfg)

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
