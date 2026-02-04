package email_confirmation_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/mewstcom/mewst/go/internal/config"
	handler "github.com/mewstcom/mewst/go/internal/handler/email_confirmation"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/templates"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// setupTestHandler はテスト用のハンドラーとテストデータをセットアップする
func setupTestHandler(t *testing.T, db *sql.DB, tx *sql.Tx) (*handler.Handler, *config.Config) {
	t.Helper()

	cfg := &config.Config{
		Env:             "test",
		Port:            "3000",
		Domain:          "localhost",
		CookieDomain:    "localhost",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	// トランザクションを使用するリポジトリを作成
	userRepo := repository.NewUserRepository(db).WithTx(tx)
	actorRepo := repository.NewActorRepository(db).WithTx(tx)
	sessionRepo := repository.NewSessionRepository(db).WithTx(tx)
	emailConfirmRepo := repository.NewEmailConfirmationRepository(db).WithTx(tx)

	sessionMgr := session.NewManager(sessionRepo, actorRepo, userRepo, cfg)

	// UseCaseの初期化
	markEmailAsConfirmedUC := usecase.NewMarkEmailAsConfirmedUsecase(emailConfirmRepo)

	h := handler.NewHandler(cfg, sessionMgr, emailConfirmRepo, markEmailAsConfirmedUC)

	return h, cfg
}

func TestNew_WithValidEmailConfirmationID(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, cfg := setupTestHandler(t, db, tx)

	// メール確認レコードを作成
	emailConfirmID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("123456").
		WithEvent("password_reset").
		Build()

	// CSRFトークンをコンテキストに設定
	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, cfg)

	req := httptest.NewRequest(http.MethodGet, "/email_confirmation", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmID.String(),
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.New(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusOK)
	}

	// レスポンスに確認コード入力フォームが含まれているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "csrf_token") {
		t.Error("CSRFトークンがフォームに含まれていません")
	}
	if !strings.Contains(body, `name="code"`) {
		t.Error("確認コードフィールドがフォームに含まれていません")
	}
}

func TestNew_WithoutEmailConfirmationID(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, cfg := setupTestHandler(t, db, tx)

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, cfg)

	// クッキーなしでリクエスト
	req := httptest.NewRequest(http.MethodGet, "/email_confirmation", nil)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.New(rr, req)

	// ルートへリダイレクトされることを検証
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("リダイレクト先が不正: got %v, want /", location)
	}
}

func TestNew_WithInvalidEmailConfirmationID(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, cfg := setupTestHandler(t, db, tx)

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, cfg)

	// 無効なUUIDでリクエスト
	req := httptest.NewRequest(http.MethodGet, "/email_confirmation", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: "invalid-uuid",
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.New(rr, req)

	// ルートへリダイレクトされることを検証
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("リダイレクト先が不正: got %v, want /", location)
	}
}

func TestNew_WithExpiredEmailConfirmation(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, cfg := setupTestHandler(t, db, tx)

	// 期限切れのメール確認レコードを作成（16分前）
	expiredTime := time.Now().Add(-16 * time.Minute)
	emailConfirmID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("123456").
		WithEvent("password_reset").
		WithCreatedAt(expiredTime).
		Build()

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, cfg)

	req := httptest.NewRequest(http.MethodGet, "/email_confirmation", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmID.String(),
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.New(rr, req)

	// ルートへリダイレクトされることを検証
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
	h, cfg := setupTestHandler(t, db, tx)

	// メール確認レコードを作成
	emailConfirmID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("123456").
		WithEvent("password_reset").
		Build()

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, cfg)

	// フォームデータを作成
	form := url.Values{}
	form.Set("code", "123456")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmID.String(),
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	// リダイレクトを検証
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}

	// パスワード編集ページへリダイレクトを検証
	location := rr.Header().Get("Location")
	if location != "/password/edit" {
		t.Errorf("リダイレクト先が不正: got %v, want /password/edit", location)
	}

	// フラッシュメッセージクッキーが設定されているか確認
	cookies := rr.Result().Cookies()
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

func TestCreate_IncorrectCode(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, cfg := setupTestHandler(t, db, tx)

	// メール確認レコードを作成
	emailConfirmID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("123456").
		WithEvent("password_reset").
		Build()

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, cfg)

	// 間違った確認コードで送信
	form := url.Values{}
	form.Set("code", "999999")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmID.String(),
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	// ステータスコードを検証（422 Unprocessable Entity）
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// エラーメッセージが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "間違っている") || !strings.Contains(body, "有効期限") {
		t.Error("エラーメッセージが表示されていません")
	}
}

func TestCreate_EmptyCode(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, cfg := setupTestHandler(t, db, tx)

	// メール確認レコードを作成
	emailConfirmID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("123456").
		WithEvent("password_reset").
		Build()

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, cfg)

	// 空の確認コードで送信
	form := url.Values{}
	form.Set("code", "")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmID.String(),
	})
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

func TestCreate_InvalidCodeFormat(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, cfg := setupTestHandler(t, db, tx)

	// メール確認レコードを作成
	emailConfirmID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("123456").
		WithEvent("password_reset").
		Build()

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, cfg)

	// 不正な形式の確認コードで送信
	form := url.Values{}
	form.Set("code", "abc123")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmID.String(),
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// バリデーションエラーが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "6桁の数字") {
		t.Error("確認コード形式のバリデーションエラーメッセージが表示されていません")
	}
}

func TestCreate_WithoutEmailConfirmationID(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, cfg := setupTestHandler(t, db, tx)

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, cfg)

	// フォームデータを作成
	form := url.Values{}
	form.Set("code", "123456")
	form.Set("csrf_token", "test-csrf-token")

	// クッキーなしでリクエスト
	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	// ルートへリダイレクトされることを検証
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("リダイレクト先が不正: got %v, want /", location)
	}
}

func TestCreate_SignUpEvent_RedirectsToAccountsNew(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, cfg := setupTestHandler(t, db, tx)

	// sign_upイベントのメール確認レコードを作成
	emailConfirmID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("123456").
		WithEvent("sign_up").
		Build()

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, cfg)

	// フォームデータを作成
	form := url.Values{}
	form.Set("code", "123456")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmID.String(),
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	// リダイレクトを検証
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}

	// /accounts/newへリダイレクトを検証
	location := rr.Header().Get("Location")
	if location != "/accounts/new" {
		t.Errorf("リダイレクト先が不正: got %v, want /accounts/new", location)
	}

	// フラッシュメッセージクッキーが設定されているか確認
	cookies := rr.Result().Cookies()
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

func TestCreate_WithExpiredEmailConfirmation(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, cfg := setupTestHandler(t, db, tx)

	// 期限切れのメール確認レコードを作成（16分前）
	expiredTime := time.Now().Add(-16 * time.Minute)
	emailConfirmID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("123456").
		WithEvent("password_reset").
		WithCreatedAt(expiredTime).
		Build()

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, cfg)

	// 正しい確認コードで送信
	form := url.Values{}
	form.Set("code", "123456")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmID.String(),
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	// ステータスコードを検証（422 Unprocessable Entity）
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// エラーメッセージが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "間違っている") || !strings.Contains(body, "有効期限") {
		t.Error("期限切れエラーメッセージが表示されていません")
	}
}
