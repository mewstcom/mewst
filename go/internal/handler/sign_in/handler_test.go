package sign_in_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/mewstcom/mewst/go/internal/config"
	handler "github.com/mewstcom/mewst/go/internal/handler/sign_in"
	"github.com/mewstcom/mewst/go/internal/middleware"
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

func (m *mockTurnstile) Verify(ctx context.Context, token string) (bool, error) {
	return m.shouldSucceed, nil
}

// setupTestHandler はテスト用のハンドラーとテストデータをセットアップする
func setupTestHandler(t *testing.T, tx *sql.Tx, turnstileSuccess bool) (*handler.Handler, *config.Config) {
	t.Helper()

	cfg := testutil.NewTestConfig(t)

	// トランザクションを使用するリポジトリを作成
	userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
	actorRepo := repository.NewActorRepository(testutil.QueriesWithTx(tx))
	sessionRepo := repository.NewSessionRepository(testutil.QueriesWithTx(tx))

	sessionMgr := session.NewManager(sessionRepo, actorRepo, userRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)
	signInValidator := validator.NewSignInCreateValidator(userRepo)
	signInUC := usecase.NewCreateSignInUsecase(signInValidator, actorRepo, sessionRepo)
	turnstile := &mockTurnstile{shouldSucceed: turnstileSuccess}

	h := handler.NewHandler(cfg, sessionMgr, flashMgr, signInUC, turnstile)

	return h, cfg
}

// createTestUser はテスト用のユーザー、プロフィール、アクターを作成する
func createTestUser(t *testing.T, tx *sql.Tx, email string) {
	t.Helper()

	// ユーザーを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail(email).
		Build()

	// プロフィールを作成
	profileID := testutil.NewProfileBuilder(t, tx).
		WithAtname("testuser").
		Build()

	// アクターを作成
	testutil.NewActorBuilder(t, tx).
		WithUserID(userID).
		WithProfileID(profileID).
		Build()
}

func TestNew(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx, true)

	// CSRFトークンをコンテキストに設定
	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	req := httptest.NewRequest(http.MethodGet, "/sign_in", nil)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.New(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusOK)
	}

	// レスポンスにログインフォームが含まれているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "csrf_token") {
		t.Error("CSRFトークンがフォームに含まれていません")
	}
	if !strings.Contains(body, "email") {
		t.Error("メールアドレスフィールドがフォームに含まれていません")
	}
	if !strings.Contains(body, "password") {
		t.Error("パスワードフィールドがフォームに含まれていません")
	}
}

func TestCreate_Success(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx, true)

	// テストユーザーを作成
	testEmail := "test-success@example.com"
	createTestUser(t, tx, testEmail)

	// CSRFトークンをコンテキストに設定
	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	// フォームデータを作成
	form := url.Values{}
	form.Set("email", testEmail)
	form.Set("password", "password") // ビルダーのデフォルトパスワード
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/sign_in", strings.NewReader(form.Encode()))
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
}

func TestCreate_InvalidEmail(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx, true)

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	// 不正なメールアドレスでログイン試行
	form := url.Values{}
	form.Set("email", "invalid-email")
	form.Set("password", "password")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/sign_in", strings.NewReader(form.Encode()))
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

func TestCreate_UserNotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx, true)

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	// 存在しないユーザーでログイン試行
	form := url.Values{}
	form.Set("email", "nonexistent@example.com")
	form.Set("password", "password")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/sign_in", strings.NewReader(form.Encode()))
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
	if !strings.Contains(body, "ログインに失敗しました") {
		t.Error("認証エラーメッセージが表示されていません")
	}
}

func TestCreate_WrongPassword(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx, true)

	// テストユーザーを作成
	testEmail := "test-wrongpass@example.com"
	createTestUser(t, tx, testEmail)

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	// 間違ったパスワードでログイン試行
	form := url.Values{}
	form.Set("email", testEmail)
	form.Set("password", "wrongpassword")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/sign_in", strings.NewReader(form.Encode()))
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
	if !strings.Contains(body, "ログインに失敗しました") {
		t.Error("認証エラーメッセージが表示されていません")
	}
}

func TestCreate_TurnstileFailed(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx, false) // Turnstile検証失敗

	// テストユーザーを作成
	testEmail := "test-turnstile@example.com"
	createTestUser(t, tx, testEmail)

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	form := url.Values{}
	form.Set("email", testEmail)
	form.Set("password", "password")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "invalid-token")

	req := httptest.NewRequest(http.MethodPost, "/sign_in", strings.NewReader(form.Encode()))
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

func TestCreate_EmptyFields(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx, true)

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	// 空のフィールドでログイン試行
	form := url.Values{}
	form.Set("email", "")
	form.Set("password", "")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/sign_in", strings.NewReader(form.Encode()))
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

func TestNew_WithBackParameter(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx, true)

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	// backパラメータ付きでリクエスト
	req := httptest.NewRequest(http.MethodGet, "/sign_in?back=/settings", nil)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.New(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusOK)
	}

	// レスポンスにback hiddenフィールドが含まれているか確認
	body := rr.Body.String()
	if !strings.Contains(body, `name="back"`) {
		t.Error("back hiddenフィールドがフォームに含まれていません")
	}
	if !strings.Contains(body, `value="/settings"`) {
		t.Error("back hiddenフィールドの値が正しくありません")
	}

	// サインアップリンクにbackパラメータが含まれているか確認 (URLクエリエスケープ済み)
	if !strings.Contains(body, `/sign_up?back=%2Fsettings`) {
		t.Error("サインアップリンクにbackパラメータが含まれていません")
	}
}

func TestCreate_SuccessWithBackParameter(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx, true)

	// テストユーザーを作成
	testEmail := "test-back-success@example.com"
	createTestUser(t, tx, testEmail)

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	// backパラメータ付きでログイン
	form := url.Values{}
	form.Set("email", testEmail)
	form.Set("password", "password")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")
	form.Set("back", "/settings")

	req := httptest.NewRequest(http.MethodPost, "/sign_in", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	// リダイレクトを検証
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}

	// リダイレクト先がbackパラメータの値であることを検証
	location := rr.Header().Get("Location")
	if location != "/settings" {
		t.Errorf("リダイレクト先が不正: got %v, want /settings", location)
	}
}

func TestCreate_SuccessWithInvalidBackParameter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		backURL string
		wantURL string
	}{
		{
			name:    "絶対URL (外部サイト)",
			backURL: "https://evil.com",
			wantURL: "/",
		},
		{
			name:    "プロトコル相対URL (外部サイト)",
			backURL: "//evil.com",
			wantURL: "/",
		},
		{
			name:    "空文字",
			backURL: "",
			wantURL: "/",
		},
		{
			name:    "正常な相対パス",
			backURL: "/profile",
			wantURL: "/profile",
		},
		{
			name:    "クエリパラメータ付き相対パス",
			backURL: "/search?q=test",
			wantURL: "/search?q=test",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			h, _ := setupTestHandler(t, tx, true)

			// テストユーザーを作成
			testEmail := "test-invalid-back-" + tc.name + "@example.com"
			createTestUser(t, tx, testEmail)

			ctx := context.Background()
			ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

			form := url.Values{}
			form.Set("email", testEmail)
			form.Set("password", "password")
			form.Set("csrf_token", "test-csrf-token")
			form.Set("cf-turnstile-response", "test-token")
			form.Set("back", tc.backURL)

			req := httptest.NewRequest(http.MethodPost, "/sign_in", strings.NewReader(form.Encode()))
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
			if location != tc.wantURL {
				t.Errorf("リダイレクト先が不正: got %v, want %v", location, tc.wantURL)
			}
		})
	}
}

func TestCreate_ValidationErrorWithBackParameter(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx, true)

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	// backパラメータ付きで無効なメールアドレスでログイン試行
	form := url.Values{}
	form.Set("email", "invalid-email")
	form.Set("password", "password")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")
	form.Set("back", "/settings")

	req := httptest.NewRequest(http.MethodPost, "/sign_in", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	// ステータスコードを検証 (422 Unprocessable Entity)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// レスポンスにback hiddenフィールドが保持されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, `name="back"`) {
		t.Error("back hiddenフィールドがフォームに保持されていません")
	}
	if !strings.Contains(body, `value="/settings"`) {
		t.Error("back hiddenフィールドの値が保持されていません")
	}
}
