package session

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/mewstcom/mewst/internal/config"
)

func TestFormErrors(t *testing.T) {
	t.Parallel()

	t.Run("NewFormErrorsは空のエラーを返す", func(t *testing.T) {
		t.Parallel()

		fe := NewFormErrors()
		if fe.HasErrors() {
			t.Error("NewFormErrors().HasErrors() = true, want false")
		}
	})

	t.Run("AddGlobalErrorでグローバルエラーを追加できる", func(t *testing.T) {
		t.Parallel()

		fe := NewFormErrors()
		fe.AddGlobalError("エラー1")
		fe.AddGlobalError("エラー2")

		if !fe.HasErrors() {
			t.Error("HasErrors() = false, want true")
		}
		if len(fe.Global) != 2 {
			t.Errorf("len(Global) = %d, want 2", len(fe.Global))
		}
		if fe.Global[0] != "エラー1" {
			t.Errorf("Global[0] = %q, want %q", fe.Global[0], "エラー1")
		}
	})

	t.Run("AddFieldErrorでフィールドエラーを追加できる", func(t *testing.T) {
		t.Parallel()

		fe := NewFormErrors()
		fe.AddFieldError("email", "メールアドレスを入力してください")
		fe.AddFieldError("email", "メールアドレスの形式が不正です")
		fe.AddFieldError("password", "パスワードを入力してください")

		if !fe.HasErrors() {
			t.Error("HasErrors() = false, want true")
		}
		if !fe.HasFieldError("email") {
			t.Error("HasFieldError(email) = false, want true")
		}
		if !fe.HasFieldError("password") {
			t.Error("HasFieldError(password) = false, want true")
		}
		if fe.HasFieldError("unknown") {
			t.Error("HasFieldError(unknown) = true, want false")
		}
	})

	t.Run("GetFieldErrorsでフィールドエラーを取得できる", func(t *testing.T) {
		t.Parallel()

		fe := NewFormErrors()
		fe.AddFieldError("email", "エラー1")
		fe.AddFieldError("email", "エラー2")

		errors := fe.GetFieldErrors("email")
		if len(errors) != 2 {
			t.Errorf("len(GetFieldErrors(email)) = %d, want 2", len(errors))
		}

		notFound := fe.GetFieldErrors("unknown")
		if notFound != nil {
			t.Errorf("GetFieldErrors(unknown) = %v, want nil", notFound)
		}
	})

	t.Run("FieldErrorsでフィールドエラーをスライスで取得できる", func(t *testing.T) {
		t.Parallel()

		fe := NewFormErrors()
		fe.AddFieldError("email", "メールエラー")
		fe.AddFieldError("password", "パスワードエラー")

		errors := fe.FieldErrors()
		if len(errors) != 2 {
			t.Errorf("len(FieldErrors()) = %d, want 2", len(errors))
		}
	})
}

func TestFlashContext(t *testing.T) {
	t.Parallel()

	t.Run("SetFlashToContextとGetFlashFromContextでフラッシュを保存・取得できる", func(t *testing.T) {
		t.Parallel()

		flash := &Flash{
			Type:    FlashSuccess,
			Message: "成功しました",
		}

		ctx := context.Background()
		ctx = SetFlashToContext(ctx, flash)

		got := GetFlashFromContext(ctx)
		if got == nil {
			t.Fatal("GetFlashFromContext() = nil, want flash")
		}
		if got.Type != FlashSuccess {
			t.Errorf("Flash.Type = %v, want %v", got.Type, FlashSuccess)
		}
		if got.Message != "成功しました" {
			t.Errorf("Flash.Message = %q, want %q", got.Message, "成功しました")
		}
	})

	t.Run("GetFlashFromContextはフラッシュがない場合nilを返す", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		got := GetFlashFromContext(ctx)
		if got != nil {
			t.Errorf("GetFlashFromContext() = %v, want nil", got)
		}
	})
}

func TestFormErrorsContext(t *testing.T) {
	t.Parallel()

	t.Run("SetFormErrorsToContextとGetFormErrorsFromContextでエラーを保存・取得できる", func(t *testing.T) {
		t.Parallel()

		errors := NewFormErrors()
		errors.AddGlobalError("グローバルエラー")
		errors.AddFieldError("email", "メールエラー")

		ctx := context.Background()
		ctx = SetFormErrorsToContext(ctx, errors)

		got := GetFormErrorsFromContext(ctx)
		if got == nil {
			t.Fatal("GetFormErrorsFromContext() = nil, want errors")
		}
		if len(got.Global) != 1 {
			t.Errorf("len(Global) = %d, want 1", len(got.Global))
		}
		if !got.HasFieldError("email") {
			t.Error("HasFieldError(email) = false, want true")
		}
	})

	t.Run("GetFormErrorsFromContextはエラーがない場合nilを返す", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		got := GetFormErrorsFromContext(ctx)
		if got != nil {
			t.Errorf("GetFormErrorsFromContext() = %v, want nil", got)
		}
	})
}

func TestManager_FlashCookie(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		CookieDomain:    "example.com",
		SessionSecure:   true,
		SessionHTTPOnly: true,
	}

	manager := NewManager(nil, nil, nil, cfg)

	t.Run("SetFlashCookieでフラッシュクッキーを設定できる", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		manager.SetFlashCookie(rr, req, FlashSuccess, "ログインしました")

		cookies := rr.Result().Cookies()
		if len(cookies) != 2 {
			t.Fatalf("SetFlashCookie() set %d cookies, want 2", len(cookies))
		}

		// メッセージクッキーを確認
		var messageCookie, typeCookie *http.Cookie
		for _, c := range cookies {
			switch c.Name {
			case FlashCookieName:
				messageCookie = c
			case FlashTypeCookieName:
				typeCookie = c
			}
		}

		if messageCookie == nil {
			t.Fatal("FlashCookieName cookie not found")
		}
		// メッセージはURLエンコードされている
		expectedEncoded := url.QueryEscape("ログインしました")
		if messageCookie.Value != expectedEncoded {
			t.Errorf("messageCookie.Value = %q, want %q", messageCookie.Value, expectedEncoded)
		}
		if messageCookie.MaxAge != 60 {
			t.Errorf("messageCookie.MaxAge = %d, want 60", messageCookie.MaxAge)
		}

		if typeCookie == nil {
			t.Fatal("FlashTypeCookieName cookie not found")
		}
		if typeCookie.Value != string(FlashSuccess) {
			t.Errorf("typeCookie.Value = %q, want %q", typeCookie.Value, FlashSuccess)
		}
	})

	t.Run("GetFlashFromCookieでフラッシュを取得し削除できる", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		// クッキーにはURLエンコードされた値を設定
		req.AddCookie(&http.Cookie{
			Name:  FlashCookieName,
			Value: url.QueryEscape("テストメッセージ"),
		})
		req.AddCookie(&http.Cookie{
			Name:  FlashTypeCookieName,
			Value: string(FlashInfo),
		})

		rr := httptest.NewRecorder()
		flash := manager.GetFlashFromCookie(rr, req)

		if flash == nil {
			t.Fatal("GetFlashFromCookie() = nil, want flash")
		}
		if flash.Type != FlashInfo {
			t.Errorf("flash.Type = %v, want %v", flash.Type, FlashInfo)
		}
		// デコードされたメッセージを返す
		if flash.Message != "テストメッセージ" {
			t.Errorf("flash.Message = %q, want %q", flash.Message, "テストメッセージ")
		}

		// 削除クッキーが設定されているか確認
		cookies := rr.Result().Cookies()
		if len(cookies) != 2 {
			t.Fatalf("GetFlashFromCookie() set %d delete cookies, want 2", len(cookies))
		}
		for _, c := range cookies {
			if c.MaxAge != -1 {
				t.Errorf("Delete cookie MaxAge = %d, want -1", c.MaxAge)
			}
		}
	})

	t.Run("GetFlashFromCookieはフラッシュがない場合nilを返す", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		flash := manager.GetFlashFromCookie(rr, req)
		if flash != nil {
			t.Errorf("GetFlashFromCookie() = %v, want nil", flash)
		}
	})

	t.Run("GetFlashFromCookieはメッセージクッキーのみの場合nilを返す", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{
			Name:  FlashCookieName,
			Value: url.QueryEscape("テストメッセージ"),
		})
		// タイプクッキーがない

		rr := httptest.NewRecorder()
		flash := manager.GetFlashFromCookie(rr, req)
		if flash != nil {
			t.Errorf("GetFlashFromCookie() = %v, want nil", flash)
		}
	})
}

func TestFlashTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		flashType FlashType
		expected  string
	}{
		{FlashSuccess, "success"},
		{FlashError, "error"},
		{FlashInfo, "info"},
		{FlashWarning, "warning"},
	}

	for _, tt := range tests {
		t.Run(string(tt.flashType), func(t *testing.T) {
			t.Parallel()

			if string(tt.flashType) != tt.expected {
				t.Errorf("FlashType = %q, want %q", string(tt.flashType), tt.expected)
			}
		})
	}
}
