package accounts_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/mewstcom/mewst/go/internal/config"
	handler "github.com/mewstcom/mewst/go/internal/handler/accounts"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/ratelimit"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/templates"
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
	profileRepo := repository.NewProfileRepository(db).WithTx(tx)
	userProfileRepo := repository.NewUserProfileRepository(db).WithTx(tx)
	actorRepo := repository.NewActorRepository(db).WithTx(tx)
	sessionRepo := repository.NewSessionRepository(db).WithTx(tx)
	emailConfirmRepo := repository.NewEmailConfirmationRepository(db).WithTx(tx)
	rateLimitRepo := repository.NewRateLimitRepository(db).WithTx(tx)

	sessionMgr := session.NewManager(sessionRepo, actorRepo, userRepo, cfg)
	createAccountUC := usecase.NewCreateAccountUsecase(db, userRepo, profileRepo, userProfileRepo, actorRepo)
	createSessionUC := usecase.NewCreateSessionUsecase(actorRepo, sessionRepo)
	getSucceededEmailConfirmationUC := usecase.NewGetSucceededEmailConfirmationUsecase(emailConfirmRepo)
	turnstile := &mockTurnstile{shouldSucceed: turnstileSuccess}
	rateLimiter := ratelimit.NewLimiter(rateLimitRepo)
	accountsValidator := validator.NewAccountsCreateValidator(userRepo, profileRepo)

	h := handler.NewHandler(cfg, sessionMgr, getSucceededEmailConfirmationUC, createAccountUC, createSessionUC, turnstile, rateLimiter, accountsValidator)

	return h, cfg
}

// createVerifiedEmailConfirmation は確認済みのメール確認レコードを作成し、そのIDをクッキーに設定する
func createVerifiedEmailConfirmation(t *testing.T, tx *sql.Tx, email string, req *http.Request) {
	t.Helper()

	ecID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail(email).
		WithEvent("sign_up").
		WithSucceededAt(time.Now()).
		Build()

	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: ecID.String(),
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, cfg := setupTestHandler(t, db, tx, true)

	// 確認済みメール確認を作成
	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, cfg)

	req := httptest.NewRequest(http.MethodGet, "/accounts/new", nil)
	req = req.WithContext(ctx)

	createVerifiedEmailConfirmation(t, tx, "newuser@example.com", req)

	rr := httptest.NewRecorder()
	h.New(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusOK)
	}

	// レスポンスにフォームが含まれているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "csrf_token") {
		t.Error("CSRFトークンがフォームに含まれていません")
	}
	if !strings.Contains(body, "atname") {
		t.Error("atnameフィールドがフォームに含まれていません")
	}
	if !strings.Contains(body, "password") {
		t.Error("passwordフィールドがフォームに含まれていません")
	}
	if !strings.Contains(body, "newuser@example.com") {
		t.Error("メールアドレスがフォームに表示されていません")
	}
}

func TestNew_WithoutEmailConfirmation(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, cfg := setupTestHandler(t, db, tx, true)

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, cfg)

	// メール確認クッキーなしでリクエスト
	req := httptest.NewRequest(http.MethodGet, "/accounts/new", nil)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	h.New(rr, req)

	// トップページにリダイレクトされるべき
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("リダイレクト先が不正: got %v, want /", location)
	}
}

func TestCreate_Success(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, cfg := setupTestHandler(t, db, tx, true)

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, cfg)

	// フォームデータを作成
	form := url.Values{}
	form.Set("atname", "accttest")
	form.Set("password", "password123")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/accounts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)

	createVerifiedEmailConfirmation(t, tx, "accounts-success@example.com", req)

	rr := httptest.NewRecorder()
	h.Create(rr, req)

	// リダイレクトを検証
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}

	// リダイレクト先を検証
	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("リダイレクト先が不正: got %v, want /", location)
	}

	// セッションクッキーが設定されているか確認
	cookies := rr.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == session.CookieName {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Error("セッションクッキーが設定されていません")
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

	// email_confirmation_idクッキーが削除されているか確認
	var ecCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == session.EmailConfirmationCookieName {
			ecCookie = c
			break
		}
	}
	if ecCookie == nil {
		t.Error("email_confirmation_idクッキーの削除応答がありません")
	} else if ecCookie.MaxAge > 0 {
		t.Error("email_confirmation_idクッキーが削除されていません")
	}
}

func TestCreate_WithoutEmailConfirmation(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, cfg := setupTestHandler(t, db, tx, true)

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, cfg)

	form := url.Values{}
	form.Set("atname", "newuser")
	form.Set("password", "password123")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	// メール確認クッキーなしでリクエスト
	req := httptest.NewRequest(http.MethodPost, "/accounts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	h.Create(rr, req)

	// トップページにリダイレクトされるべき
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("リダイレクト先が不正: got %v, want /", location)
	}
}

func TestCreate_RateLimitExceeded(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, cfg := setupTestHandler(t, db, tx, true)

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, cfg)

	// レート制限を超過するためにリクエストを5回送信（制限: 5回/分）
	for i := 0; i < 5; i++ {
		form := url.Values{}
		form.Set("atname", "ratelimit")
		form.Set("password", "password123")
		form.Set("csrf_token", "test-csrf-token")
		form.Set("cf-turnstile-response", "test-token")

		req := httptest.NewRequest(http.MethodPost, "/accounts", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req = req.WithContext(ctx)

		email := fmt.Sprintf("ratelimit-%d@example.com", i)
		createVerifiedEmailConfirmation(t, tx, email, req)

		rr := httptest.NewRecorder()
		h.Create(rr, req)
	}

	// 6回目のリクエストでレート制限超過
	form := url.Values{}
	form.Set("atname", "ratelimit")
	form.Set("password", "password123")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/accounts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)

	createVerifiedEmailConfirmation(t, tx, "ratelimit-final@example.com", req)

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

func TestCreate_TurnstileFailed(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, cfg := setupTestHandler(t, db, tx, false) // Turnstile検証失敗

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, cfg)

	form := url.Values{}
	form.Set("atname", "newuser")
	form.Set("password", "password123")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "invalid-token")

	req := httptest.NewRequest(http.MethodPost, "/accounts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)

	createVerifiedEmailConfirmation(t, tx, "newuser@example.com", req)

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

func TestCreate_ValidationError_EmptyAtname(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, cfg := setupTestHandler(t, db, tx, true)

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, cfg)

	form := url.Values{}
	form.Set("atname", "")
	form.Set("password", "password123")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/accounts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)

	createVerifiedEmailConfirmation(t, tx, "newuser@example.com", req)

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

func TestCreate_ValidationError_DuplicateAtname(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, cfg := setupTestHandler(t, db, tx, true)

	// 既存のプロフィールを作成
	testutil.NewProfileBuilder(t, tx).
		WithAtname("existinguser").
		Build()

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, cfg)

	form := url.Values{}
	form.Set("atname", "existinguser")
	form.Set("password", "password123")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/accounts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)

	createVerifiedEmailConfirmation(t, tx, "newuser@example.com", req)

	rr := httptest.NewRecorder()
	h.Create(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// 重複エラーメッセージが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "既に使用されています") {
		t.Error("アットネーム重複エラーメッセージが表示されていません")
	}
}

func TestCreate_ValidationError_DuplicateEmail(t *testing.T) {
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

	form := url.Values{}
	form.Set("atname", "newuser")
	form.Set("password", "password123")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/accounts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)

	// 既存のメールアドレスで確認済みメール確認を作成
	createVerifiedEmailConfirmation(t, tx, "existing@example.com", req)

	rr := httptest.NewRecorder()
	h.Create(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// 重複エラーメッセージが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "既に登録されています") {
		t.Error("メールアドレス重複エラーメッセージが表示されていません")
	}
}
