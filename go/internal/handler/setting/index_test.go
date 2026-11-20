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

// newSettingHandlerは設定メニューのHandlerを構築する。
func newSettingHandler(t *testing.T) *setting.Handler {
	t.Helper()

	return setting.NewHandler(testutil.NewTestConfig(t))
}

// newIndexRequestはGET /settingsのリクエストを組み立てる。contextには
// 本番でCSRF / RequireAuthミドルウェアが渡すもの (ロケール、ログアウトフォームが
// 送信するCSRFトークン、ログイン中のactorとプロフィール) を載せる。プロフィール
// はnavbarのプロフィールリンクを駆動する。
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

// settingsNavは設定メニューの <nav> 要素を返す。メニューについての検証が、
// 共通レイアウト内の別のリンクで満たされてしまわないようにするため。
func settingsNav(t *testing.T, body string, label string) string {
	t.Helper()

	start := `<nav aria-label="` + label + `">`
	navIdx := strings.Index(body, start)
	if navIdx < 0 {
		t.Fatal("設定メニューのnavがありません")
	}

	endOffset := strings.Index(body[navIdx:], `</nav>`)
	if endOffset < 0 {
		t.Fatal("設定メニューのnavに閉じタグがありません")
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
		t.Errorf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	if contentType := rr.Header().Get("Content-Type"); !strings.Contains(contentType, "text/html") {
		t.Errorf("Content-Typeが不正: 実測値 = %v、期待値 = text/html", contentType)
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
		// ログアウトボタンはアウトラインのbasecoatボタンで、その送信はネイティブ
		// 確認ダイアログでガードされる。RenderAttributesはonclickの値をHTMLエスケープ
		// するため、確認文言は "confirm(" の断片とは分けて検証する。
		`class="btn rounded-full"`,
		`data-variant="outline"`,
		"confirm(",
		"ログアウトしますか？",
		`aria-label="設定メニュー"`,
		// caret-right-regular固有のpath片。Indexは未知のアイコン名を
		// info-regularにフォールバックし、両者はviewBox="0 0 256 256" を共有するため、
		// caret固有のpathを確認することでフォールバックではなく解決されたことを保証する。
		"M181.66,133.66l-80,80",
	}
	for _, want := range checks {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていません", want)
		}
	}

	// 設定メニューのリンクは、ラベル付き <nav> 内のリストに置く。
	nav := settingsNav(t, body, "設定メニュー")
	for _, want := range []string{
		`<ul class="flex flex-col">`,
		`href="/settings/profile"`,
		`href="/settings/user"`,
		`href="/settings/email"`,
		`href="/settings/export"`,
	} {
		if !strings.Contains(nav, want) {
			t.Errorf("設定メニューのnavに%qが含まれていません", want)
		}
	}
	if got := strings.Count(nav, `<li>`); got != 4 {
		t.Errorf("設定メニューのli数 = %d、期待値 = 4", got)
	}
	if got := strings.Count(nav, `aria-hidden="true"`); got != 4 {
		t.Errorf("装飾キャレットのaria-hidden数 = %d、期待値 = 4", got)
	}
	if got := strings.Count(nav, "M181.66,133.66l-80,80"); got != 4 {
		t.Errorf("caret-right-regularのpath数 = %d、期待値 = 4", got)
	}

	// このページは認証後のnavbar付きレイアウト (layouts.Default) で描画するため、
	// トップnavbar (PC) とボトムnavbar (モバイル) が5項目メニューを描画する。navbar
	// 専用リンク (検索) と、注入したatnameから生成されるプロフィールリンクを検証し、
	// 設定コンテンツの周囲に両navbarが描画されることを確認する。
	navbarChecks := []struct {
		link string
		want int
	}{
		{link: `href="/search"`, want: 2},
		{link: `href="/@alice"`, want: 2},
	}
	for _, check := range navbarChecks {
		if got := strings.Count(body, check.link); got != check.want {
			t.Errorf("navbarの%qの数 = %d、期待値 = %d", check.link, got, check.want)
		}
	}

	// 設定はnavbarの5項目に含まれないため、navbarはアクティブ項目なし
	// (NavbarItemNone) で描画する。したがってアクティブの塗りつぶしアイコンのfill
	// 上書きは一切現れない。
	if got := strings.Count(body, "[&_.content]:fill-foreground"); got != 0 {
		t.Errorf("アクティブ表示のfillクラス数 = %d、期待値 = 0 (設定はnavbar項目を持たない)", got)
	}
}

// TestIndex_PlacesTheExportEntryLastは、エクスポート項目がメニューの
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
		t.Errorf("ステータスコードが不正: 実測値 = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	nav := settingsNav(t, rr.Body.String(), "設定メニュー")

	// エクスポート項目はアカウント設定の各行の後、最後に置く。
	if exportIdx, emailIdx := strings.Index(nav, `href="/settings/export"`), strings.Index(nav, `href="/settings/email"`); exportIdx < emailIdx {
		t.Errorf("エクスポート項目の位置 = %d, メールアドレス項目の位置 = %d (エクスポートは末尾に置く)", exportIdx, emailIdx)
	}

	// リンクテキストはスクリーンリーダーのリンク一覧で単独で意味を成す必要が
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
			name:           "日本語",
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
			name:           "英語",
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
					t.Errorf("%sのレスポンスに%qが含まれていません", tt.locale, want)
				}
			}
		})
	}
}
