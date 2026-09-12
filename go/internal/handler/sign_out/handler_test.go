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

// setupTestHandler sets up the handler under test.
//
// [Ja] setupTestHandler はテスト用のハンドラーをセットアップする。
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

// findCookie returns the cookie with the given name from the response, or nil.
//
// [Ja] findCookie はレスポンスから指定名の Cookie を返す (無ければ nil)。
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
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}

	deleted, err := sessionRepo.FindByToken(ctx, token)
	if err != nil {
		t.Fatalf("FindByToken() error = %v", err)
	}
	if deleted != nil {
		t.Error("DB のセッションレコードが削除されていません")
	}
}

func TestDelete_SessionDeletionFailure(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, _ := setupTestHandler(t, tx)

	// Roll the transaction back before the request so every query through it
	// fails with sql.ErrTxDone. This is how the test forces the session deletion
	// to fail without reaching for a mock repository.
	//
	// [Ja] リクエスト前にトランザクションをロールバックし、これを経由するクエリを
	// すべて sql.ErrTxDone で失敗させる。モックの repository を持ち込まずに
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

	// Sign-out must still succeed: the cookie is cleared, the flash is set and
	// the user is redirected even though the session row could not be deleted.
	//
	// [Ja] セッションレコードを削除できなくてもログアウト自体は成立しなければ
	// ならない。Cookie が削除され、フラッシュが設定され、リダイレクトされる。
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

			// Decode the same cookie through FlashManager to verify its type and
			// message.
			//
			// [Ja] FlashManager 経由で同じ Cookie をデコードしてタイプとメッセージを検証する。
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
			// Verify the handler honours the locale in ctx and writes the message
			// translated for that locale into the cookie. The expectation is built
			// through i18n.T rather than a literal so it follows wording changes in
			// the translation files.
			//
			// [Ja] handler が ctx の locale を尊重し、ロケールごとに翻訳されたメッセージが
			// Cookie に書かれていることを確認する。翻訳ファイルの文言変更に追従できるよう、
			// 期待値はリテラルではなく i18n.T 経由で生成する。
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

// signOutCSRFToken is an arbitrary token shared by a request's CSRF cookie and
// form field. The middleware only checks that the two match, so any non-empty
// value works.
//
// [Ja] signOutCSRFToken はリクエストの CSRF Cookie とフォームフィールドで共有する
// 任意のトークン。ミドルウェアは両者の一致だけを見るため、非空なら何でもよい。
const signOutCSRFToken = "test-csrf-token-1234567890"

// TestDelete_CSRFMiddleware_RejectsRequestWithoutToken verifies that fronting the
// sign-out handler with the CSRF middleware (as main.go now does for the
// /sign_out group) blocks a forged POST that carries no CSRF token, before the
// handler runs. A CSRF forgery rides the victim's session cookie but cannot read
// the token, so the session cookie is present while the token is absent.
//
// [Ja] TestDelete_CSRFMiddleware_RejectsRequestWithoutToken は、ログアウトハンドラーの
// 前段に CSRF ミドルウェアを置いたとき (main.go が /sign_out グループに対して行うのと
// 同じ)、CSRF トークンを持たない偽造 POST をハンドラー実行前に遮断することを検証する。
// CSRF 偽造は被害者のセッション Cookie に便乗するがトークンは読めないため、セッション
// Cookie はあってもトークンは無い。
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
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusForbidden)
	}

	// The middleware must block before the handler runs, so none of sign-out's
	// side effects (the flash message) are produced.
	//
	// [Ja] ミドルウェアはハンドラー実行前に遮断しなければならないため、ログアウトの
	// 副作用 (フラッシュメッセージ) は一切生じない。
	if findCookie(rr.Result().Cookies(), session.FlashCookieName) != nil {
		t.Error("403 のはずがフラッシュメッセージクッキーが設定されています")
	}
}

// TestDelete_CSRFMiddleware_AcceptsRequestWithValidToken verifies that the same
// wiring lets a POST through when its form token matches the CSRF cookie, so the
// sign-out completes with the usual redirect. This is the token flow the Go
// /settings page produces: the CSRF middleware issues the cookie and the page
// embeds the matching token in the sign-out form.
//
// [Ja] TestDelete_CSRFMiddleware_AcceptsRequestWithValidToken は、同じ配線でフォーム
// トークンが CSRF Cookie と一致する POST を通し、ログアウトが通常のリダイレクトで
// 完了することを検証する。これは Go 版 /settings ページが生む トークンの流れである。
// CSRF ミドルウェアが Cookie を発行し、ページがログアウトフォームに一致するトークンを
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
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusFound)
	}
	if location := rr.Header().Get("Location"); location != "/" {
		t.Errorf("リダイレクト先が不正: got %v, want /", location)
	}
	if findCookie(rr.Result().Cookies(), session.FlashCookieName) == nil {
		t.Error("フラッシュメッセージクッキーが設定されていません")
	}
}
