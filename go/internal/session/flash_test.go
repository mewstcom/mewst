package session_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mewstcom/mewst/go/internal/session"
)

func TestFlashManager_SetSuccess(t *testing.T) {
	t.Parallel()

	fm := session.NewFlashManager(".example.com", true, true)

	rr := httptest.NewRecorder()
	fm.SetSuccess(rr, "ログインしました")

	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Cookieの数が不正: got %d, want 1", len(cookies))
	}

	cookie := cookies[0]
	if cookie.Name != session.FlashCookieName {
		t.Errorf("Cookie名が不正: got %s, want %s", cookie.Name, session.FlashCookieName)
	}
	if cookie.Value == "" {
		t.Error("Cookie値が空です")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookie)

	rr2 := httptest.NewRecorder()
	flash := fm.GetFlash(rr2, req)

	if flash == nil {
		t.Fatal("フラッシュメッセージがnilです")
	}
	if flash.Type != session.FlashSuccess {
		t.Errorf("タイプが不正: got %s, want %s", flash.Type, session.FlashSuccess)
	}
	if flash.Message != "ログインしました" {
		t.Errorf("メッセージが不正: got %s, want ログインしました", flash.Message)
	}
}

func TestFlashManager_SetError(t *testing.T) {
	t.Parallel()

	fm := session.NewFlashManager(".example.com", true, true)

	rr := httptest.NewRecorder()
	fm.SetError(rr, "エラーが発生しました")

	cookie := rr.Result().Cookies()[0]
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookie)

	rr2 := httptest.NewRecorder()
	flash := fm.GetFlash(rr2, req)

	if flash == nil {
		t.Fatal("フラッシュメッセージがnilです")
	}
	if flash.Type != session.FlashError {
		t.Errorf("タイプが不正: got %s, want %s", flash.Type, session.FlashError)
	}
	if flash.Message != "エラーが発生しました" {
		t.Errorf("メッセージが不正: got %s, want エラーが発生しました", flash.Message)
	}
}

func TestFlashManager_SetWarning(t *testing.T) {
	t.Parallel()

	fm := session.NewFlashManager(".example.com", true, true)

	rr := httptest.NewRecorder()
	fm.SetWarning(rr, "注意が必要です")

	cookie := rr.Result().Cookies()[0]
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookie)

	rr2 := httptest.NewRecorder()
	flash := fm.GetFlash(rr2, req)

	if flash == nil {
		t.Fatal("フラッシュメッセージがnilです")
	}
	if flash.Type != session.FlashWarning {
		t.Errorf("タイプが不正: got %s, want %s", flash.Type, session.FlashWarning)
	}
	if flash.Message != "注意が必要です" {
		t.Errorf("メッセージが不正: got %s, want 注意が必要です", flash.Message)
	}
}

func TestFlashManager_SetInfo(t *testing.T) {
	t.Parallel()

	fm := session.NewFlashManager(".example.com", true, true)

	rr := httptest.NewRecorder()
	fm.SetInfo(rr, "お知らせがあります")

	cookie := rr.Result().Cookies()[0]
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookie)

	rr2 := httptest.NewRecorder()
	flash := fm.GetFlash(rr2, req)

	if flash == nil {
		t.Fatal("フラッシュメッセージがnilです")
	}
	if flash.Type != session.FlashInfo {
		t.Errorf("タイプが不正: got %s, want %s", flash.Type, session.FlashInfo)
	}
	if flash.Message != "お知らせがあります" {
		t.Errorf("メッセージが不正: got %s, want お知らせがあります", flash.Message)
	}
}

func TestFlashManager_GetFlash(t *testing.T) {
	t.Parallel()

	fm := session.NewFlashManager(".example.com", true, true)

	t.Run("Cookieがない場合はnilを返すこと", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		flash := fm.GetFlash(rr, req)

		if flash != nil {
			t.Errorf("フラッシュメッセージがnilではありません: %+v", flash)
		}
	})

	t.Run("フラッシュ取得後にCookieが削除されること", func(t *testing.T) {
		t.Parallel()

		rr := httptest.NewRecorder()
		fm.SetSuccess(rr, "テストメッセージ")
		cookie := rr.Result().Cookies()[0]

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(cookie)

		rr2 := httptest.NewRecorder()
		flash := fm.GetFlash(rr2, req)

		if flash == nil {
			t.Fatal("フラッシュメッセージがnilです")
		}

		deleteCookies := rr2.Result().Cookies()
		if len(deleteCookies) != 1 {
			t.Fatalf("削除用Cookieの数が不正: got %d, want 1", len(deleteCookies))
		}

		if deleteCookies[0].MaxAge != -1 {
			t.Errorf("MaxAgeが不正: got %d, want -1", deleteCookies[0].MaxAge)
		}
	})

	t.Run("不正なBase64の場合はnilを返してCookieを削除すること", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{
			Name:  session.FlashCookieName,
			Value: "!!! invalid base64 !!!",
		})

		rr := httptest.NewRecorder()
		flash := fm.GetFlash(rr, req)

		if flash != nil {
			t.Errorf("フラッシュメッセージがnilではありません: %+v", flash)
		}

		deleteCookies := rr.Result().Cookies()
		if len(deleteCookies) != 1 {
			t.Fatalf("削除用Cookieの数が不正: got %d, want 1", len(deleteCookies))
		}
		if deleteCookies[0].MaxAge != -1 {
			t.Errorf("MaxAgeが不正: got %d, want -1", deleteCookies[0].MaxAge)
		}
	})
}

func TestFlashManager_CookieAttributes(t *testing.T) {
	t.Parallel()

	t.Run("HttpOnlyがfalseであること", func(t *testing.T) {
		t.Parallel()

		fm := session.NewFlashManager(".example.com", true, true)
		rr := httptest.NewRecorder()
		fm.SetSuccess(rr, "テスト")

		cookie := rr.Result().Cookies()[0]
		if cookie.HttpOnly {
			t.Error("HttpOnlyフラグがtrueです（falseであるべき）")
		}
	})

	t.Run("SameSiteがLaxであること", func(t *testing.T) {
		t.Parallel()

		fm := session.NewFlashManager(".example.com", true, true)
		rr := httptest.NewRecorder()
		fm.SetSuccess(rr, "テスト")

		cookie := rr.Result().Cookies()[0]
		if cookie.SameSite != http.SameSiteLaxMode {
			t.Errorf("SameSiteが不正: got %v, want %v", cookie.SameSite, http.SameSiteLaxMode)
		}
	})
}

func TestFlashManager_Middleware(t *testing.T) {
	t.Parallel()

	fm := session.NewFlashManager(".example.com", true, true)

	t.Run("Cookieにフラッシュメッセージがあるとcontextに設定されること", func(t *testing.T) {
		t.Parallel()

		// フラッシュをセットして cookie を取り出す
		rr := httptest.NewRecorder()
		fm.SetSuccess(rr, "ようこそ")
		cookie := rr.Result().Cookies()[0]

		var observed *session.FlashMessage
		next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			observed = session.FlashFromContext(r.Context())
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(cookie)
		rr2 := httptest.NewRecorder()
		fm.Middleware(next).ServeHTTP(rr2, req)

		if observed == nil {
			t.Fatal("contextからフラッシュが取得できませんでした")
		}
		if observed.Type != session.FlashSuccess {
			t.Errorf("タイプが不正: got %s, want %s", observed.Type, session.FlashSuccess)
		}
		if observed.Message != "ようこそ" {
			t.Errorf("メッセージが不正: got %s, want ようこそ", observed.Message)
		}
	})

	t.Run("Cookieがなければcontextには何も入らないこと", func(t *testing.T) {
		t.Parallel()

		called := false
		next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			called = true
			if flash := session.FlashFromContext(r.Context()); flash != nil {
				t.Errorf("フラッシュが残っています: %+v", flash)
			}
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		fm.Middleware(next).ServeHTTP(rr, req)

		if !called {
			t.Fatal("next ハンドラーが呼ばれませんでした")
		}
	})
}

func TestFlashFromContext_NoValue(t *testing.T) {
	t.Parallel()

	if flash := session.FlashFromContext(context.Background()); flash != nil {
		t.Errorf("フラッシュメッセージがnilではありません: %+v", flash)
	}
}
