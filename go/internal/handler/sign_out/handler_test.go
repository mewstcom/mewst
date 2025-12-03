package sign_out_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mewstcom/mewst/internal/config"
	handler "github.com/mewstcom/mewst/internal/handler/sign_out"
	"github.com/mewstcom/mewst/internal/repository"
	"github.com/mewstcom/mewst/internal/session"
	"github.com/mewstcom/mewst/internal/templates"
	"github.com/mewstcom/mewst/internal/testutil"
)

// setupTestHandler はテスト用のハンドラーをセットアップする
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

	sessionMgr := session.NewManager(sessionRepo, actorRepo, userRepo, cfg)

	h := handler.NewHandler(cfg, sessionMgr)

	return h, cfg
}

func TestDelete_Success(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, cfg := setupTestHandler(t, db, tx)

	// コンテキストをセットアップ
	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, cfg)

	// セッションクッキー付きのリクエストを作成
	req := httptest.NewRequest(http.MethodDelete, "/sign_out", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.CookieName,
		Value: "test-session-token",
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	// リダイレクトを検証
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}

	// リダイレクト先を検証
	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("リダイレクト先が不正: got %v, want /", location)
	}

	// セッションクッキーが削除されているか確認
	cookies := rr.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == session.CookieName {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Error("セッションクッキーがレスポンスに含まれていません")
	} else if sessionCookie.MaxAge != -1 {
		t.Errorf("セッションクッキーのMaxAgeが不正: got %v, want -1", sessionCookie.MaxAge)
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

func TestDelete_WithoutSession(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, cfg := setupTestHandler(t, db, tx)

	// コンテキストをセットアップ
	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, cfg)

	// セッションクッキーなしのリクエストを作成
	req := httptest.NewRequest(http.MethodDelete, "/sign_out", nil)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	// リダイレクトを検証（セッションがなくてもエラーにならない）
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}

	// リダイレクト先を検証
	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("リダイレクト先が不正: got %v, want /", location)
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

func TestDelete_FlashMessage_Japanese(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, cfg := setupTestHandler(t, db, tx)

	// 日本語ロケールでコンテキストをセットアップ
	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "ja")
	ctx = templates.WithConfig(ctx, cfg)

	req := httptest.NewRequest(http.MethodDelete, "/sign_out", nil)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	// フラッシュメッセージの内容を検証
	cookies := rr.Result().Cookies()
	var flashCookie *http.Cookie
	var flashTypeCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == session.FlashCookieName {
			flashCookie = c
		}
		if c.Name == session.FlashTypeCookieName {
			flashTypeCookie = c
		}
	}

	if flashCookie == nil {
		t.Fatal("フラッシュメッセージクッキーが見つかりません")
	}

	// フラッシュタイプがsuccessであることを確認
	if flashTypeCookie == nil {
		t.Fatal("フラッシュタイプクッキーが見つかりません")
	}
	if flashTypeCookie.Value != string(session.FlashSuccess) {
		t.Errorf("フラッシュタイプが不正: got %v, want %v", flashTypeCookie.Value, session.FlashSuccess)
	}
}

func TestDelete_FlashMessage_English(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	h, cfg := setupTestHandler(t, db, tx)

	// 英語ロケールでコンテキストをセットアップ
	ctx := context.Background()
	ctx = templates.WithLocale(ctx, "en")
	ctx = templates.WithConfig(ctx, cfg)

	req := httptest.NewRequest(http.MethodDelete, "/sign_out", nil)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	// フラッシュメッセージの内容を検証
	cookies := rr.Result().Cookies()
	var flashTypeCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == session.FlashTypeCookieName {
			flashTypeCookie = c
			break
		}
	}

	if flashTypeCookie == nil {
		t.Fatal("フラッシュタイプクッキーが見つかりません")
	}
	if flashTypeCookie.Value != string(session.FlashSuccess) {
		t.Errorf("フラッシュタイプが不正: got %v, want %v", flashTypeCookie.Value, session.FlashSuccess)
	}
}
