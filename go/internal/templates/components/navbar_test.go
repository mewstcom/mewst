package components_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/templates/components"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// activeFillClass / inactiveFillClassはアクティブ / 非アクティブの
// メニューアイコンに適用されるfill上書きクラス。
const (
	activeFillClass   = "[&_.content]:fill-foreground"
	inactiveFillClass = "[&_.content]:fill-gray-500 dark:[&_.content]:fill-gray-300"
)

// anchorSegmentは指定したhrefを含む <a> 要素の範囲を切り出して返す。
// その内側のアイコンを検証するために使う。
func anchorSegment(html, href string) string {
	marker := `href="` + href + `"`
	i := strings.Index(html, marker)
	if i < 0 {
		return ""
	}
	start := strings.LastIndex(html[:i], "<a ")
	end := strings.Index(html[i:], "</a>")
	if start < 0 || end < 0 {
		return ""
	}
	return html[start : i+end]
}

func renderComponent(t *testing.T, c templ.Component) string {
	t.Helper()

	var buf bytes.Buffer
	if err := c.Render(context.Background(), &buf); err != nil {
		t.Fatalf("描画に失敗: %v", err)
	}
	return buf.String()
}

func TestNavbarMenu_Links(t *testing.T) {
	t.Parallel()

	navbar := viewmodel.NewNavbar(&model.Profile{Atname: "alice"}, viewmodel.NavbarItemNew)
	html := renderComponent(t, components.NavbarMenu(navbar, ""))

	wantHrefs := []string{
		`href="/home"`,
		`href="/search"`,
		`href="/new"`,
		`href="/notifications"`,
		`href="/@alice"`,
	}
	for _, want := range wantHrefs {
		if !strings.Contains(html, want) {
			t.Errorf("NavbarMenuの出力にリンク%qが含まれていない", want)
		}
	}

	// 各項目はアイコンを1つ描画するため、アイコンは5個ちょうどになる。
	if got := strings.Count(html, "<svg"); got != 5 {
		t.Errorf("svgの件数 = %d、期待値 = 5", got)
	}

	// ピル状コンテナは共有のクリーム色背景の上で独立した面として見えるよう
	// 白背景を使う。
	if !strings.Contains(html, "bg-white") {
		t.Error("NavbarMenuの出力に白いピル型の背景が含まれていない")
	}

	// ダークモードのhover背景は、アクティブ / 非アクティブ双方のアイコンと
	// 3:1のコントラストを保てる暗さを維持する。
	if !strings.Contains(html, "dark:hover:bg-stone-700") {
		t.Error("NavbarMenuの出力にダークモードのホバー背景が含まれていない")
	}
}

func TestNavbarMenu_ActiveHighlight(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		activeItem viewmodel.NavbarItem
		activeHref string
	}{
		{name: "homeがアクティブ", activeItem: viewmodel.NavbarItemHome, activeHref: "/home"},
		{name: "searchがアクティブ", activeItem: viewmodel.NavbarItemSearch, activeHref: "/search"},
		{name: "newがアクティブ", activeItem: viewmodel.NavbarItemNew, activeHref: "/new"},
		{name: "notificationがアクティブ", activeItem: viewmodel.NavbarItemNotification, activeHref: "/notifications"},
		{name: "profileがアクティブ", activeItem: viewmodel.NavbarItemProfile, activeHref: "/@alice"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			navbar := viewmodel.NewNavbar(&model.Profile{Atname: "alice"}, tt.activeItem)
			html := renderComponent(t, components.NavbarMenu(navbar, ""))

			// アクティブ (塗りつぶしで現在ページとしてマーク) は1項目だけで、
			// 残り4項目は非アクティブ。
			if got := strings.Count(html, activeFillClass); got != 1 {
				t.Errorf("アクティブの塗りクラスの件数 = %d、期待値 = 1", got)
			}
			if got := strings.Count(html, inactiveFillClass); got != 4 {
				t.Errorf("非アクティブの塗りクラスの件数 = %d、期待値 = 4", got)
			}
			if got := strings.Count(html, `aria-current="page"`); got != 1 {
				t.Errorf(`aria-current="page"の件数 = %d、期待値 = 1`, got)
			}

			// アクティブなfillと現在ページの状態は、アクティブ項目のリンク内に
			// 存在しなければならない。
			seg := anchorSegment(html, tt.activeHref)
			if seg == "" {
				t.Fatalf("%qへのリンクが見つからない", tt.activeHref)
			}
			if !strings.Contains(seg, activeFillClass) {
				t.Errorf("%qへのリンクがアクティブとして描画されていない", tt.activeHref)
			}
			if !strings.Contains(seg, `aria-current="page"`) {
				t.Errorf("%qへのリンクが現在のページとして示されていない", tt.activeHref)
			}
		})
	}
}

func TestNavbarMenu_AriaLabels(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), "ja")
	navbar := viewmodel.NewNavbar(&model.Profile{Atname: "alice"}, viewmodel.NavbarItemHome)

	var buf bytes.Buffer
	if err := components.NavbarMenu(navbar, "").Render(ctx, &buf); err != nil {
		t.Fatalf("描画に失敗: %v", err)
	}
	html := buf.String()

	// アイコンのみのリンクは、各リンクの目的をスクリーンリーダーが読み上げ
	// られるようaria-labelを持つ必要がある。ローカライズ済みラベルを検証する
	// ことで、i18nキーが正しいリンクに紐づいていることも確認する。
	wantLabels := []string{"ホーム", "検索", "新規投稿", "通知", "プロフィール"}
	for _, want := range wantLabels {
		if !strings.Contains(html, `aria-label="`+want+`"`) {
			t.Errorf("NavbarMenuの出力にaria-label %qが含まれていない", want)
		}
	}
}

func TestNavbarMenu_ClassName(t *testing.T) {
	t.Parallel()

	navbar := viewmodel.NewNavbar(&model.Profile{Atname: "alice"}, viewmodel.NavbarItemHome)
	html := renderComponent(t, components.NavbarMenu(navbar, "border border-slate-400"))

	if !strings.Contains(html, "border border-slate-400") {
		t.Error("NavbarMenuの出力に追加したclassNameが含まれていない")
	}
}

func TestTopNavbar(t *testing.T) {
	t.Parallel()

	navbar := viewmodel.NewNavbar(&model.Profile{Atname: "alice"}, viewmodel.NavbarItemNew)
	html := renderComponent(t, components.TopNavbar(navbar))

	// 名前を持つロゴはrootへリンクし、バーは上辺のsafe areaより下に固定して
	// lg以上でのみ表示する。
	checks := []string{`href="/"`, `aria-label="Mewst"`, "sticky", "top-safe", "bg-background", "lg:block", "hidden"}
	for _, want := range checks {
		if !strings.Contains(html, want) {
			t.Errorf("TopNavbarの出力に%qが含まれていない", want)
		}
	}

	// 5項目すべてのメニューがトップnavbarに埋め込まれている。
	if !strings.Contains(html, `href="/@alice"`) {
		t.Error("TopNavbarの出力にnavbarメニューが含まれていない")
	}
}

func TestSharedOverlaysRespectSafeAreas(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		html    string
		classes []string
	}{
		{
			name:    "スキップリンク",
			html:    renderComponent(t, components.SkipLink()),
			classes: []string{"focus:left-safe-offset-4", "focus:top-safe-offset-4"},
		},
		{
			name: "フラッシュのトースター",
			html: renderComponent(t, components.Flash(&session.FlashMessage{
				Type:    session.FlashInfo,
				Message: "safe-area marker",
			})),
			classes: []string{"pb-safe-offset-4", "px-safe-offset-4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, class := range tt.classes {
				if !strings.Contains(tt.html, class) {
					t.Errorf("コンポーネントの出力に%qが含まれていない", class)
				}
			}
		})
	}
}

func TestTopNavbar_Landmark(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), "ja")
	navbar := viewmodel.NewNavbar(&model.Profile{Atname: "alice"}, viewmodel.NavbarItemNew)

	var buf bytes.Buffer
	if err := components.TopNavbar(navbar).Render(ctx, &buf); err != nil {
		t.Fatalf("描画に失敗: %v", err)
	}
	html := buf.String()

	// ラッパーはローカライズ済みaria-labelを持つ <nav> ランドマークで、
	// 共存する2つのナビゲーション領域を区別できるようにする。
	if !strings.Contains(html, "<nav") {
		t.Error("TopNavbarの出力に<nav>ランドマークが含まれていない")
	}
	if !strings.Contains(html, `aria-label="グローバル (PC)"`) {
		t.Error("TopNavbarの出力にaria-labelが含まれていない")
	}
}

func TestBottomNavbar(t *testing.T) {
	t.Parallel()

	navbar := viewmodel.NewNavbar(&model.Profile{Atname: "alice"}, viewmodel.NavbarItemNew)
	html := renderComponent(t, components.BottomNavbar(navbar))

	// ボトムnavbarはlg未満でのみ表示し、枠線を持つ。
	checks := []string{"lg:hidden", "border border-slate-400", `href="/new"`}
	for _, want := range checks {
		if !strings.Contains(html, want) {
			t.Errorf("BottomNavbarの出力に%qが含まれていない", want)
		}
	}
}

func TestBottomNavbar_Landmark(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), "ja")
	navbar := viewmodel.NewNavbar(&model.Profile{Atname: "alice"}, viewmodel.NavbarItemNew)

	var buf bytes.Buffer
	if err := components.BottomNavbar(navbar).Render(ctx, &buf); err != nil {
		t.Fatalf("描画に失敗: %v", err)
	}
	html := buf.String()

	// ラッパーは <nav> ランドマークで、そのaria-labelはトップnavbarとは
	// 異なり、2つのナビゲーション領域を区別する。
	if !strings.Contains(html, "<nav") {
		t.Error("BottomNavbarの出力に<nav>ランドマークが含まれていない")
	}
	if !strings.Contains(html, `aria-label="グローバル (モバイル)"`) {
		t.Error("BottomNavbarの出力にaria-labelが含まれていない")
	}
}
