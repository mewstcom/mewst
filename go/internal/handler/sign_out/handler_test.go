package sign_out_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/mewstcom/mewst/go/internal/config"
	handler "github.com/mewstcom/mewst/go/internal/handler/sign_out"
	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// setupTestHandlerはテスト用のハンドラーをセットアップする。
func setupTestHandler(t *testing.T, tx *sql.Tx) (*handler.Handler, *config.Config) {
	t.Helper()

	cfg := testutil.NewTestConfig(t)

	userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
	actorRepo := repository.NewActorRepository(testutil.QueriesWithTx(tx))
	sessionRepo := repository.NewSessionRepository(testutil.QueriesWithTx(tx))

	sessionMgr := session.NewManager(sessionRepo, actorRepo, userRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	deleteSessionUC := usecase.NewDeleteSessionUsecase(sessionRepo)

	h := handler.NewHandler(sessionMgr, flashMgr, deleteSessionUC)

	return h, cfg
}

// findCookieはレスポンスから指定名のCookieを返す (無ければnil)。
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
		t.Errorf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusFound)
	}

	if location := rr.Header().Get("Location"); location != "/" {
		t.Errorf("リダイレクト先が不正: 実測値 = %v、期待値 = /", location)
	}

	cookies := rr.Result().Cookies()

	sessionCookie := findCookie(cookies, session.CookieName)
	if sessionCookie == nil {
		t.Error("セッションクッキーがレスポンスに含まれていません")
	} else if sessionCookie.MaxAge != -1 {
		t.Errorf("セッションクッキーのMaxAgeが不正: 実測値 = %v、期待値 = -1", sessionCookie.MaxAge)
	}

	if findCookie(cookies, session.FlashCookieName) == nil {
		t.Error("フラッシュメッセージクッキーが設定されていません")
	}
}

func TestDelete_DeletesSessionRecord(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx)

	const token = "stored-session-token"

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("sign-out-stored-session@example.com").
		Build()

	profileID := testutil.NewProfileBuilder(t, tx).
		WithAtname("signoutuser").
		Build()

	actorID := testutil.NewActorBuilder(t, tx).
		WithUserID(userID).
		WithProfileID(profileID).
		Build()

	testutil.NewSessionBuilder(t, tx).
		WithActorID(actorID).
		WithToken(token).
		Build()

	sessionRepo := repository.NewSessionRepository(testutil.QueriesWithTx(tx))

	ctx := context.Background()

	req := httptest.NewRequest(http.MethodDelete, "/sign_out", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.CookieName,
		Value: token,
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusFound)
	}

	deleted, err := sessionRepo.FindByToken(ctx, token)
	if err != nil {
		t.Fatalf("FindByToken()のエラー = %v", err)
	}
	if deleted != nil {
		t.Error("DBのセッションレコードが削除されていません")
	}
}

func TestDelete_SessionDeletionFailure(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx)

	// リクエスト前にトランザクションをロールバックし、これを経由するクエリを
	// すべてsql.ErrTxDoneで失敗させる。モックのrepositoryを持ち込まずに
	// セッション削除の失敗を再現するための手段。
	if err := tx.Rollback(); err != nil {
		t.Fatalf("トランザクションのロールバックに失敗: %v", err)
	}

	ctx := context.Background()

	req := httptest.NewRequest(http.MethodDelete, "/sign_out", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.CookieName,
		Value: "test-session-token",
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	// セッションレコードを削除できなくてもログアウト自体は成立しなければ
	// ならない。Cookieが削除され、フラッシュが設定され、リダイレクトされる。
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusFound)
	}

	if location := rr.Header().Get("Location"); location != "/" {
		t.Errorf("リダイレクト先が不正: 実測値 = %v、期待値 = /", location)
	}

	cookies := rr.Result().Cookies()

	sessionCookie := findCookie(cookies, session.CookieName)
	if sessionCookie == nil {
		t.Error("セッションクッキーがレスポンスに含まれていません")
	} else if sessionCookie.MaxAge != -1 {
		t.Errorf("セッションクッキーのMaxAgeが不正: 実測値 = %v、期待値 = -1", sessionCookie.MaxAge)
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
		t.Errorf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusFound)
	}

	if location := rr.Header().Get("Location"); location != "/" {
		t.Errorf("リダイレクト先が不正: 実測値 = %v、期待値 = /", location)
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

			// FlashManager経由で同じCookieをデコードしてタイプとメッセージを検証する。
			fm := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)
			verifyReq := httptest.NewRequest(http.MethodGet, "/", nil)
			verifyReq.AddCookie(flashCookie)
			flash := fm.GetFlash(httptest.NewRecorder(), verifyReq)
			if flash == nil {
				t.Fatal("フラッシュメッセージのデコードに失敗")
			}
			if flash.Type != session.FlashSuccess {
				t.Errorf("フラッシュタイプが不正: 実測値 = %v、期待値 = %v", flash.Type, session.FlashSuccess)
			}
			// handlerがctxのlocaleを尊重し、ロケールごとに翻訳されたメッセージが
			// Cookieに書かれていることを確認する。翻訳ファイルの文言変更に追従できるよう、
			// 期待値はリテラルではなくi18n.T経由で生成する。
			expectedCtx := i18n.SetLocale(context.Background(), tt.locale)
			want := i18n.T(expectedCtx, "flash_sign_out_success")
			if flash.Message != want {
				t.Errorf("フラッシュメッセージが不正: 実測値 = %q、期待値 = %q", flash.Message, want)
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
		t.Errorf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusFound)
	}
	if location := rr.Header().Get("Location"); location != "/" {
		t.Errorf("リダイレクト先が不正: 実測値 = %v、期待値 = /", location)
	}

	sessionCookie := findCookie(rr.Result().Cookies(), session.CookieName)
	if sessionCookie == nil {
		t.Error("セッションクッキーがレスポンスに含まれていません")
	} else if sessionCookie.MaxAge != -1 {
		t.Errorf("セッションクッキーのMaxAgeが不正: 実測値 = %v、期待値 = -1", sessionCookie.MaxAge)
	}
}

// signOutCSRFTokenはリクエストのCSRF Cookieとフォームフィールドで共有する
// 任意のトークン。ミドルウェアは両者の一致だけを見るため、非空なら何でもよい。
const signOutCSRFToken = "test-csrf-token-1234567890"

// TestDelete_CSRFMiddleware_RejectsRequestWithoutTokenは、ログアウトハンドラーの
// 前段にCSRFミドルウェアを置いたとき (main.goが /sign_outグループに対して行うのと
// 同じ)、CSRFトークンを持たない偽造POSTをハンドラー実行前に遮断することを検証する。
// CSRF偽造は被害者のセッションCookieに便乗するがトークンは読めないため、セッション
// Cookieはあってもトークンは無い。
func TestDelete_CSRFMiddleware_RejectsRequestWithoutToken(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, cfg := setupTestHandler(t, tx)

	protected := middleware.NewCSRF(cfg).Middleware(http.HandlerFunc(h.Delete))

	req := httptest.NewRequest(http.MethodPost, "/sign_out", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.CookieName,
		Value: "test-session-token",
	})
	rr := httptest.NewRecorder()

	protected.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusForbidden)
	}

	// ミドルウェアはハンドラー実行前に遮断しなければならないため、ログアウトの
	// 副作用 (フラッシュメッセージ) は一切生じない。
	if findCookie(rr.Result().Cookies(), session.FlashCookieName) != nil {
		t.Error("403のはずがフラッシュメッセージクッキーが設定されています")
	}
}

// TestDelete_CSRFMiddleware_AcceptsRequestWithValidTokenは、同じ配線でフォーム
// トークンがCSRF Cookieと一致するPOSTを通し、ログアウトが通常のリダイレクトで
// 完了することを検証する。これはGo版 /settingsページが生む トークンの流れである。
// CSRFミドルウェアがCookieを発行し、ページがログアウトフォームに一致するトークンを
// 埋め込む。
func TestDelete_CSRFMiddleware_AcceptsRequestWithValidToken(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, cfg := setupTestHandler(t, tx)

	protected := middleware.NewCSRF(cfg).Middleware(http.HandlerFunc(h.Delete))

	form := url.Values{}
	form.Set("csrf_token", signOutCSRFToken)

	req := httptest.NewRequest(http.MethodPost, "/sign_out", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  middleware.CSRFCookieName,
		Value: signOutCSRFToken,
	})
	req.AddCookie(&http.Cookie{
		Name:  session.CookieName,
		Value: "test-session-token",
	})
	rr := httptest.NewRecorder()

	protected.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusFound)
	}
	if location := rr.Header().Get("Location"); location != "/" {
		t.Errorf("リダイレクト先が不正: 実測値 = %v、期待値 = /", location)
	}
	if findCookie(rr.Result().Cookies(), session.FlashCookieName) == nil {
		t.Error("フラッシュメッセージクッキーが設定されていません")
	}
}
