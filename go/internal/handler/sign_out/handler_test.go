package sign_out_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mewstcom/mewst/go/internal/config"
	handler "github.com/mewstcom/mewst/go/internal/handler/sign_out"
	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

// setupTestHandler はテスト用のハンドラーをセットアップする
func setupTestHandler(t *testing.T, tx *sql.Tx) (*handler.Handler, *config.Config) {
	t.Helper()

	cfg := testutil.NewTestConfig(t)

	userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
	actorRepo := repository.NewActorRepository(testutil.QueriesWithTx(tx))
	sessionRepo := repository.NewSessionRepository(testutil.QueriesWithTx(tx))
	profileRepo := repository.NewProfileRepository(testutil.QueriesWithTx(tx))

	sessionMgr := session.NewManager(sessionRepo, actorRepo, userRepo, profileRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	h := handler.NewHandler(cfg, sessionMgr, flashMgr)

	return h, cfg
}

// findCookie はレスポンスから指定名のCookieを返す (無ければnil)
func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, c := range cookies {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func TestDelete_Success(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx)

	ctx := context.Background()

	req := httptest.NewRequest(http.MethodDelete, "/sign_out", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.CookieName,
		Value: "test-session-token",
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}

	if location := rr.Header().Get("Location"); location != "/" {
		t.Errorf("リダイレクト先が不正: got %v, want /", location)
	}

	cookies := rr.Result().Cookies()

	sessionCookie := findCookie(cookies, session.CookieName)
	if sessionCookie == nil {
		t.Error("セッションクッキーがレスポンスに含まれていません")
	} else if sessionCookie.MaxAge != -1 {
		t.Errorf("セッションクッキーのMaxAgeが不正: got %v, want -1", sessionCookie.MaxAge)
	}

	if findCookie(cookies, session.FlashCookieName) == nil {
		t.Error("フラッシュメッセージクッキーが設定されていません")
	}
}

func TestDelete_WithoutSession(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx)

	ctx := context.Background()

	req := httptest.NewRequest(http.MethodDelete, "/sign_out", nil)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}

	if location := rr.Header().Get("Location"); location != "/" {
		t.Errorf("リダイレクト先が不正: got %v, want /", location)
	}

	cookies := rr.Result().Cookies()
	if findCookie(cookies, session.FlashCookieName) == nil {
		t.Error("フラッシュメッセージクッキーが設定されていません")
	}
}

func TestDelete_FlashMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		locale string
	}{
		{name: "日本語ロケール", locale: "ja"},
		{name: "英語ロケール", locale: "en"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			h, cfg := setupTestHandler(t, tx)

			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, tt.locale)

			req := httptest.NewRequest(http.MethodDelete, "/sign_out", nil)
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			h.Delete(rr, req)

			flashCookie := findCookie(rr.Result().Cookies(), session.FlashCookieName)
			if flashCookie == nil {
				t.Fatal("フラッシュメッセージクッキーが見つかりません")
			}

			// FlashManager 経由で同じ Cookie をデコードしてタイプとメッセージを検証する
			fm := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)
			verifyReq := httptest.NewRequest(http.MethodGet, "/", nil)
			verifyReq.AddCookie(flashCookie)
			flash := fm.GetFlash(httptest.NewRecorder(), verifyReq)
			if flash == nil {
				t.Fatal("フラッシュメッセージのデコードに失敗")
			}
			if flash.Type != session.FlashSuccess {
				t.Errorf("フラッシュタイプが不正: got %v, want %v", flash.Type, session.FlashSuccess)
			}
			// handler が ctx の locale を尊重し、ロケールごとに翻訳されたメッセージが Cookie に書かれていることを確認する
			// 翻訳ファイルの文言変更に追従できるよう、期待値はリテラルではなく i18n.T 経由で生成する
			expectedCtx := i18n.SetLocale(context.Background(), tt.locale)
			want := i18n.T(expectedCtx, "flash_sign_out_success")
			if flash.Message != want {
				t.Errorf("フラッシュメッセージが不正: got %q, want %q", flash.Message, want)
			}
		})
	}
}

func TestDelete_POSTMethod(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx)

	ctx := context.Background()

	req := httptest.NewRequest(http.MethodPost, "/sign_out", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.CookieName,
		Value: "test-session-token",
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}
	if location := rr.Header().Get("Location"); location != "/" {
		t.Errorf("リダイレクト先が不正: got %v, want /", location)
	}

	sessionCookie := findCookie(rr.Result().Cookies(), session.CookieName)
	if sessionCookie == nil {
		t.Error("セッションクッキーがレスポンスに含まれていません")
	} else if sessionCookie.MaxAge != -1 {
		t.Errorf("セッションクッキーのMaxAgeが不正: got %v, want -1", sessionCookie.MaxAge)
	}
}
