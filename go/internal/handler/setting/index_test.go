package setting_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mewstcom/mewst/go/internal/handler/setting"
	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

// newSettingHandler builds a Handler for the settings menu.
//
// [Ja] newSettingHandler は設定メニューの Handler を構築する。
func newSettingHandler(t *testing.T) *setting.Handler {
	t.Helper()

	return setting.NewHandler(testutil.NewTestConfig(t))
}

// newIndexRequest builds a GET /settings request whose context carries what the
// CSRF and RequireAuth middleware supply in production: the locale, the CSRF
// token the sign-out form submits, and the signed-in actor and profile. The
// profile drives the navbar's profile link.
//
// [Ja] newIndexRequest は GET /settings のリクエストを組み立てる。context には
// 本番で CSRF / RequireAuth ミドルウェアが渡すもの (ロケール、ログアウトフォームが
// 送信する CSRF トークン、ログイン中の actor とプロフィール) を載せる。プロフィール
// は navbar のプロフィールリンクを駆動する。
func newIndexRequest(t *testing.T, locale string, owner testutil.ProfileOwner) *http.Request {
	t.Helper()

	ctx := i18n.SetLocale(context.Background(), locale)
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = middleware.SetActorToContext(ctx, &model.Actor{
		ID:        owner.ActorID,
		UserID:    owner.UserID,
		ProfileID: owner.ProfileID,
	})
	ctx = middleware.SetProfileToContext(ctx, &model.Profile{
		ID:     owner.ProfileID,
		Atname: "alice",
	})

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	return req.WithContext(ctx)
}

// settingsNav returns the settings menu's <nav> element, so assertions about
// the menu cannot be satisfied by links elsewhere in the shared layout.
//
// [Ja] settingsNav は設定メニューの <nav> 要素を返す。メニューについての検証が、
// 共通レイアウト内の別のリンクで満たされてしまわないようにするため。
func settingsNav(t *testing.T, body string, label string) string {
	t.Helper()

	start := `<nav aria-label="` + label + `">`
	navIdx := strings.Index(body, start)
	if navIdx < 0 {
		t.Fatal("設定メニューの nav がありません")
	}

	endOffset := strings.Index(body[navIdx:], `</nav>`)
	if endOffset < 0 {
		t.Fatal("設定メニューの nav に閉じタグがありません")
	}

	return body[navIdx : navIdx+endOffset+len(`</nav>`)]
}

func TestIndex(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	owner := testutil.NewProfileOwner(t, tx)
	h := newSettingHandler(t)

	rr := httptest.NewRecorder()
	h.Index(rr, newIndexRequest(t, "ja", owner))

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
		`href="/settings/export"`,
		"プロフィールの編集",
		"ユーザーの編集",
		"メールアドレスの変更",
		"ポストのエクスポート",
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
	//
	// [Ja] 設定メニューのリンクは、ラベル付き <nav> 内のリストに置く。
	nav := settingsNav(t, body, "設定メニュー")
	for _, want := range []string{
		`<ul class="flex flex-col">`,
		`href="/settings/profile"`,
		`href="/settings/user"`,
		`href="/settings/email"`,
		`href="/settings/export"`,
	} {
		if !strings.Contains(nav, want) {
			t.Errorf("設定メニューの nav に %q が含まれていません", want)
		}
	}
	if got := strings.Count(nav, `<li>`); got != 4 {
		t.Errorf("設定メニューの li 数 = %d, want 4", got)
	}
	if got := strings.Count(nav, `aria-hidden="true"`); got != 4 {
		t.Errorf("装飾キャレットの aria-hidden 数 = %d, want 4", got)
	}
	if got := strings.Count(nav, "M181.66,133.66l-80,80"); got != 4 {
		t.Errorf("caret-right-regular の path 数 = %d, want 4", got)
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

// TestIndex_PlacesTheExportEntryLast pins where the export entry sits in the
// menu and how its link reads: it is the last row, after the rows that act on
// the account itself, and its text names what the page is for rather than
// reading as a bare action.
//
// [Ja] TestIndex_PlacesTheExportEntryLast は、エクスポート項目がメニューの
// どこに置かれ、そのリンクがどう読めるかを固定する。項目はアカウント自体を扱う
// 各行の後、最後の行に置き、リンクテキストは素の操作名ではなく遷移先が何で
// あるかを示す。
func TestIndex_PlacesTheExportEntryLast(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	owner := testutil.NewProfileOwner(t, tx)
	h := newSettingHandler(t)

	rr := httptest.NewRecorder()
	h.Index(rr, newIndexRequest(t, "ja", owner))

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコードが不正: got %v, want %v", rr.Code, http.StatusOK)
	}

	nav := settingsNav(t, rr.Body.String(), "設定メニュー")

	// The export entry goes last, after the account settings rows.
	//
	// [Ja] エクスポート項目はアカウント設定の各行の後、最後に置く。
	if exportIdx, emailIdx := strings.Index(nav, `href="/settings/export"`), strings.Index(nav, `href="/settings/email"`); exportIdx < emailIdx {
		t.Errorf("エクスポート項目の位置 = %d, メールアドレス項目の位置 = %d (エクスポートは末尾に置く)", exportIdx, emailIdx)
	}

	// The link text must stand on its own in a screen reader's link list, so it
	// names the object it acts on instead of the bare verb.
	//
	// [Ja] リンクテキストはスクリーンリーダーのリンク一覧で単独で意味を成す必要が
	// あるため、素の動詞ではなく対象を含めて示す。
	if strings.Contains(nav, `>エクスポート<`) {
		t.Error("設定メニューのリンクテキストが素の「エクスポート」になっています")
	}
}

func TestIndex_Locales(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		locale         string
		title          string
		heading        string
		menuLabel      string
		profile        string
		user           string
		email          string
		export         string
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
			export:         "ポストのエクスポート",
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
			export:         "Export posts",
			signOut:        "Sign out",
			signOutConfirm: "Are you sure you want to sign out?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			owner := testutil.NewProfileOwner(t, tx)
			h := newSettingHandler(t)

			rr := httptest.NewRecorder()
			h.Index(rr, newIndexRequest(t, tt.locale, owner))

			body := rr.Body.String()
			for _, want := range []string{
				`<title>` + tt.title + ` | Mewst</title>`,
				`<h1 class="text-2xl font-semibold antialiased">` + tt.heading + `</h1>`,
				`aria-label="` + tt.menuLabel + `"`,
				tt.profile,
				tt.user,
				tt.email,
				tt.export,
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
