package setting_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/handler/setting"
	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

func newSettingHandler(t *testing.T) *setting.Handler {
	t.Helper()

	return setting.NewHandler(testutil.NewTestConfig(t))
}

func TestIndex(t *testing.T) {
	t.Parallel()

	h := newSettingHandler(t)

	// Set the CSRF token, locale, and current profile on the context the way the
	// CSRF and RequireAuth middleware do in production. The CSRF token drives the
	// sign-out form's hidden input; the profile drives the navbar's profile link.
	//
	// [Ja] CSRF トークン・ロケール・現在プロフィールを、本番で CSRF / RequireAuth
	// ミドルウェアが行うのと同じ形で context に設定する。CSRF トークンはログアウト
	// フォームの hidden input を、プロフィールは navbar のプロフィールリンクを駆動する。
	ctx := i18n.SetLocale(context.Background(), "ja")
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = middleware.SetProfileToContext(ctx, &model.Profile{
		ID:     model.ProfileID(uuid.MustParse("11111111-1111-1111-1111-111111111111")),
		Atname: "alice",
	})

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusOK)
	}

	if contentType := rr.Header().Get("Content-Type"); !strings.Contains(contentType, "text/html") {
		t.Errorf("Content-Type が不正: got %v, want text/html", contentType)
	}

	body := rr.Body.String()
	checks := []string{
		"設定",
		`<h1`,
		`href="/settings/profile"`,
		`href="/settings/user"`,
		`href="/settings/email"`,
		"プロフィールの編集",
		"ユーザーの編集",
		"メールアドレスの変更",
		`action="/sign_out"`,
		`method="POST"`,
		`name="csrf_token"`,
		`value="test-csrf-token"`,
		"ログアウト",
		// The sign-out button is an outline basecoat button whose submit is guarded
		// by a native confirm dialog. RenderAttributes HTML-escapes the onclick
		// value, so the message text is asserted separately from the "confirm("
		// fragment.
		//
		// [Ja] ログアウトボタンはアウトラインの basecoat ボタンで、その送信はネイティブ
		// 確認ダイアログでガードされる。RenderAttributes は onclick の値を HTML エスケープ
		// するため、確認文言は "confirm(" の断片とは分けて検証する。
		`class="btn rounded-full"`,
		`data-variant="outline"`,
		"confirm(",
		"ログアウトしますか？",
		`aria-label="設定メニュー"`,
		// Path fragment unique to caret-right-regular. Index falls back to
		// info-regular for unknown icon names, and both share viewBox="0 0 256 256",
		// so asserting the caret's own path proves it resolved rather than fell back.
		//
		// [Ja] caret-right-regular 固有の path 片。Index は未知のアイコン名を
		// info-regular にフォールバックし、両者は viewBox="0 0 256 256" を共有するため、
		// caret 固有の path を確認することでフォールバックではなく解決されたことを保証する。
		"M181.66,133.66l-80,80",
	}
	for _, want := range checks {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに %q が含まれていません", want)
		}
	}

	// The settings menu links must stay inside a labelled <nav> containing a list.
	// Limit the assertions to that nav so links elsewhere in the shared layout
	// cannot make this semantic-structure check pass.
	//
	// [Ja] 設定メニューのリンクは、ラベル付き <nav> 内のリストに置く。この nav の
	// 範囲だけを検証し、共通レイアウト内の別リンクによる偽陽性を防ぐ。
	const settingsNavStart = `<nav aria-label="設定メニュー">`
	navIdx := strings.Index(body, settingsNavStart)
	if navIdx < 0 {
		t.Error("設定メニューの nav がありません")
	} else {
		navEndOffset := strings.Index(body[navIdx:], `</nav>`)
		if navEndOffset < 0 {
			t.Error("設定メニューの nav に閉じタグがありません")
		} else {
			settingsNav := body[navIdx : navIdx+navEndOffset+len(`</nav>`)]
			for _, want := range []string{
				`<ul class="flex flex-col">`,
				`href="/settings/profile"`,
				`href="/settings/user"`,
				`href="/settings/email"`,
			} {
				if !strings.Contains(settingsNav, want) {
					t.Errorf("設定メニューの nav に %q が含まれていません", want)
				}
			}
			if got := strings.Count(settingsNav, `<li>`); got != 3 {
				t.Errorf("設定メニューの li 数 = %d, want 3", got)
			}
			if got := strings.Count(settingsNav, `aria-hidden="true"`); got != 3 {
				t.Errorf("装飾キャレットの aria-hidden 数 = %d, want 3", got)
			}
			if got := strings.Count(settingsNav, "M181.66,133.66l-80,80"); got != 3 {
				t.Errorf("caret-right-regular の path 数 = %d, want 3", got)
			}
		}
	}

	// The page renders on the authenticated navbar layout (layouts.Default), so
	// the top navbar (desktop) and bottom navbar (mobile) render the five-item
	// menu. Assert a navbar-only link (search) and the profile link built from the
	// injected atname to confirm both navbars render around the settings content.
	//
	// [Ja] このページは認証後の navbar 付きレイアウト (layouts.Default) で描画するため、
	// トップ navbar (PC) とボトム navbar (モバイル) が 5 項目メニューを描画する。navbar
	// 専用リンク (検索) と、注入した atname から生成されるプロフィールリンクを検証し、
	// 設定コンテンツの周囲に両 navbar が描画されることを確認する。
	navbarChecks := []struct {
		link string
		want int
	}{
		{link: `href="/search"`, want: 2},
		{link: `href="/@alice"`, want: 2},
	}
	for _, check := range navbarChecks {
		if got := strings.Count(body, check.link); got != check.want {
			t.Errorf("navbar の %q の数 = %d, want %d", check.link, got, check.want)
		}
	}

	// Settings is not one of the navbar's five items, so the navbar renders with no
	// active item (NavbarItemNone). The active filled-icon fill override must
	// therefore be absent entirely.
	//
	// [Ja] 設定は navbar の 5 項目に含まれないため、navbar はアクティブ項目なし
	// (NavbarItemNone) で描画する。したがってアクティブの塗りつぶしアイコンの fill
	// 上書きは一切現れない。
	if got := strings.Count(body, "[&_.content]:fill-foreground"); got != 0 {
		t.Errorf("アクティブ表示の fill クラス数 = %d, want 0 (設定は navbar 項目を持たない)", got)
	}
}

func TestIndex_Locales(t *testing.T) {
	t.Parallel()

	h := newSettingHandler(t)

	tests := []struct {
		name           string
		locale         string
		title          string
		heading        string
		menuLabel      string
		profile        string
		user           string
		email          string
		signOut        string
		signOutConfirm string
	}{
		{
			name:           "Japanese",
			locale:         "ja",
			title:          "設定",
			heading:        "設定",
			menuLabel:      "設定メニュー",
			profile:        "プロフィールの編集",
			user:           "ユーザーの編集",
			email:          "メールアドレスの変更",
			signOut:        "ログアウト",
			signOutConfirm: "ログアウトしますか？",
		},
		{
			name:           "English",
			locale:         "en",
			title:          "Settings",
			heading:        "Settings",
			menuLabel:      "Settings menu",
			profile:        "Edit profile",
			user:           "Edit user",
			email:          "Change email",
			signOut:        "Sign out",
			signOutConfirm: "Are you sure you want to sign out?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := i18n.SetLocale(context.Background(), tt.locale)
			ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")

			req := httptest.NewRequest(http.MethodGet, "/settings", nil)
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			h.Index(rr, req)

			body := rr.Body.String()
			for _, want := range []string{
				`<title>` + tt.title + ` | Mewst</title>`,
				`<h1 class="text-2xl font-semibold antialiased">` + tt.heading + `</h1>`,
				`aria-label="` + tt.menuLabel + `"`,
				tt.profile,
				tt.user,
				tt.email,
				tt.signOut,
				tt.signOutConfirm,
			} {
				if !strings.Contains(body, want) {
					t.Errorf("%s のレスポンスに %q が含まれていません", tt.locale, want)
				}
			}
		})
	}
}
