package password_test

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
	handler "github.com/mewstcom/mewst/go/internal/handler/password"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/validator"
)

// setupTestHandler はテスト用のハンドラーとテストデータをセットアップする
func setupTestHandler(t *testing.T, db *sql.DB, tx *sql.Tx) (*handler.Handler, *config.Config) {
	t.Helper()

	cfg := testutil.NewTestConfig(t)

	// トランザクションを使用するリポジトリを作成
	userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
	actorRepo := repository.NewActorRepository(testutil.QueriesWithTx(tx))
	sessionRepo := repository.NewSessionRepository(testutil.QueriesWithTx(tx))
	emailConfirmRepo := repository.NewEmailConfirmationRepository(testutil.QueriesWithTx(tx))

	sessionMgr := session.NewManager(sessionRepo, actorRepo, userRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)
	getSucceededEmailConfirmationUC := usecase.NewGetSucceededEmailConfirmationUsecase(emailConfirmRepo)
	passwordUpdateValidator := validator.NewPasswordUpdateValidator()
	updatePasswordUC := usecase.NewUpdatePasswordUsecase(passwordUpdateValidator, userRepo)

	h := handler.NewHandler(cfg, sessionMgr, flashMgr, getSucceededEmailConfirmationUC, updatePasswordUC)

	return h, cfg
}

func TestEdit_WithValidSucceededEmailConfirmation(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, _ := setupTestHandler(t, db, tx)

	// 確認済みのメール確認レコードを作成
	now := time.Now()
	emailConfirmID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("123456").
		WithEvent("password_reset").
		WithSucceededAt(now).
		Build()

	// CSRFトークンをコンテキストに設定
	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	req := httptest.NewRequest(http.MethodGet, "/password/edit", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmID.String(),
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Edit(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusOK)
	}

	// レスポンスにパスワード入力フォームが含まれているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "csrf_token") {
		t.Error("CSRFトークンがフォームに含まれていません")
	}
	if !strings.Contains(body, `name="password"`) {
		t.Error("パスワードフィールドがフォームに含まれていません")
	}
	if !strings.Contains(body, `_method`) {
		t.Error("メソッドオーバーライドがフォームに含まれていません")
	}
}

func TestEdit_WithoutEmailConfirmationID(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, _ := setupTestHandler(t, db, tx)

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	// クッキーなしでリクエスト
	req := httptest.NewRequest(http.MethodGet, "/password/edit", nil)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Edit(rr, req)

	// ルートへリダイレクトされることを検証
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("リダイレクト先が不正: got %v, want /", location)
	}
}

func TestEdit_WithInvalidEmailConfirmationID(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, _ := setupTestHandler(t, db, tx)

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	// 無効なUUIDでリクエスト
	req := httptest.NewRequest(http.MethodGet, "/password/edit", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: "invalid-uuid",
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Edit(rr, req)

	// ルートへリダイレクトされることを検証
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("リダイレクト先が不正: got %v, want /", location)
	}
}

func TestEdit_WithUnsuccessfulEmailConfirmation(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, _ := setupTestHandler(t, db, tx)

	// 未確認のメール確認レコードを作成（succeeded_atがNULL）
	emailConfirmID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("123456").
		WithEvent("password_reset").
		Build()

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	req := httptest.NewRequest(http.MethodGet, "/password/edit", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmID.String(),
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Edit(rr, req)

	// ルートへリダイレクトされることを検証
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("リダイレクト先が不正: got %v, want /", location)
	}
}

func TestEdit_WithMismatchedEvent(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, _ := setupTestHandler(t, db, tx)

	// password_reset 以外のイベント（sign_up）の確認済みレコードを作成
	emailConfirmID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("123456").
		WithEvent("sign_up").
		WithSucceededAt(time.Now()).
		Build()

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	req := httptest.NewRequest(http.MethodGet, "/password/edit", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmID.String(),
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Edit(rr, req)

	// イベント種別が異なるためルートへリダイレクトされることを検証
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("リダイレクト先が不正: got %v, want /", location)
	}
}

func TestUpdate_Success(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, _ := setupTestHandler(t, db, tx)

	// ユーザーを作成
	testutil.NewUserBuilder(t, tx).
		WithEmail("test@example.com").
		Build()

	// 確認済みのメール確認レコードを作成
	now := time.Now()
	emailConfirmID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("123456").
		WithEvent("password_reset").
		WithSucceededAt(now).
		Build()

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	// フォームデータを作成
	form := url.Values{}
	form.Set("password", "newpassword123")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPatch, "/password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmID.String(),
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	// リダイレクトを検証
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}

	// ログインページへリダイレクトを検証
	location := rr.Header().Get("Location")
	if location != "/sign_in" {
		t.Errorf("リダイレクト先が不正: got %v, want /sign_in", location)
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

	// メール確認IDクッキーが削除されているか確認
	var emailConfirmCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == session.EmailConfirmationCookieName {
			emailConfirmCookie = c
			break
		}
	}
	if emailConfirmCookie == nil || emailConfirmCookie.MaxAge != -1 {
		t.Error("メール確認IDクッキーが正しく削除されていません")
	}
}

func TestUpdate_EmptyPassword(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, _ := setupTestHandler(t, db, tx)

	// 確認済みのメール確認レコードを作成
	now := time.Now()
	emailConfirmID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("123456").
		WithEvent("password_reset").
		WithSucceededAt(now).
		Build()

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	// 空のパスワードで送信
	form := url.Values{}
	form.Set("password", "")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPatch, "/password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmID.String(),
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Update(rr, req)

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

func TestUpdate_ShortPassword(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, _ := setupTestHandler(t, db, tx)

	// 確認済みのメール確認レコードを作成
	now := time.Now()
	emailConfirmID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("123456").
		WithEvent("password_reset").
		WithSucceededAt(now).
		Build()

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	// 短すぎるパスワードで送信（8文字未満）
	form := url.Values{}
	form.Set("password", "short")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPatch, "/password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmID.String(),
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// バリデーションエラーが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "8文字以上") {
		t.Error("パスワード長のバリデーションエラーメッセージが表示されていません")
	}
}

func TestUpdate_WithoutEmailConfirmationID(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, _ := setupTestHandler(t, db, tx)

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	// フォームデータを作成
	form := url.Values{}
	form.Set("password", "newpassword123")
	form.Set("csrf_token", "test-csrf-token")

	// クッキーなしでリクエスト
	req := httptest.NewRequest(http.MethodPatch, "/password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	// ルートへリダイレクトされることを検証
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("リダイレクト先が不正: got %v, want /", location)
	}
}

func TestUpdate_WithUnsuccessfulEmailConfirmation(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, _ := setupTestHandler(t, db, tx)

	// 未確認のメール確認レコードを作成（succeeded_atがNULL）
	emailConfirmID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("123456").
		WithEvent("password_reset").
		Build()

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	// フォームデータを作成
	form := url.Values{}
	form.Set("password", "newpassword123")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPatch, "/password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmID.String(),
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	// ルートへリダイレクトされることを検証
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("リダイレクト先が不正: got %v, want /", location)
	}
}

func TestUpdate_WithMismatchedEvent(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, _ := setupTestHandler(t, db, tx)

	// password_reset 以外のイベント（sign_up）の確認済みレコードを作成
	emailConfirmID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("123456").
		WithEvent("sign_up").
		WithSucceededAt(time.Now()).
		Build()

	ctx := context.Background()
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

	form := url.Values{}
	form.Set("password", "newpassword123")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPatch, "/password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmID.String(),
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	// イベント種別が異なるためルートへリダイレクトされることを検証
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("リダイレクト先が不正: got %v, want /", location)
	}
}
