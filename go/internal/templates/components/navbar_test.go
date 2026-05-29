package components_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/templates/components"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// activeFillClass / inactiveFillClass are the fill overrides applied to the
// active and inactive menu icons respectively.
//
// [Ja] activeFillClass / inactiveFillClass はアクティブ / 非アクティブの
// メニューアイコンに適用される fill 上書きクラス。
const (
	activeFillClass   = "[&_.content]:fill-foreground"
	inactiveFillClass = "[&_.content]:fill-gray-400"
)

// anchorSegment returns the substring of html covering the <a> element whose
// markup contains the given href, so a test can assert on its inner icon.
//
// [Ja] anchorSegment は指定した href を含む <a> 要素の範囲を切り出して返す。
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
		t.Fatalf("failed to render: %v", err)
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
			t.Errorf("NavbarMenu output missing link %q", want)
		}
	}

	// Each item renders one icon, so exactly five icons should be present.
	// [Ja] 各項目はアイコンを 1 つ描画するため、アイコンは 5 個ちょうどになる。
	if got := strings.Count(html, "<svg"); got != 5 {
		t.Errorf("svg count = %d, want 5", got)
	}
}

func TestNavbarMenu_ActiveHighlight(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		activeItem viewmodel.NavbarItem
		activeHref string
	}{
		{name: "home がアクティブ", activeItem: viewmodel.NavbarItemHome, activeHref: "/home"},
		{name: "search がアクティブ", activeItem: viewmodel.NavbarItemSearch, activeHref: "/search"},
		{name: "new がアクティブ", activeItem: viewmodel.NavbarItemNew, activeHref: "/new"},
		{name: "notification がアクティブ", activeItem: viewmodel.NavbarItemNotification, activeHref: "/notifications"},
		{name: "profile がアクティブ", activeItem: viewmodel.NavbarItemProfile, activeHref: "/@alice"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			navbar := viewmodel.NewNavbar(&model.Profile{Atname: "alice"}, tt.activeItem)
			html := renderComponent(t, components.NavbarMenu(navbar, ""))

			// Exactly one item is active (filled), the other four are inactive.
			// [Ja] アクティブ (塗りつぶし) は 1 項目だけで、残り 4 項目は非アクティブ。
			if got := strings.Count(html, activeFillClass); got != 1 {
				t.Errorf("active fill class count = %d, want 1", got)
			}
			if got := strings.Count(html, inactiveFillClass); got != 4 {
				t.Errorf("inactive fill class count = %d, want 4", got)
			}

			// The active fill must belong to the active item's link.
			// [Ja] アクティブな fill はアクティブ項目のリンク内に存在しなければならない。
			seg := anchorSegment(html, tt.activeHref)
			if seg == "" {
				t.Fatalf("anchor for %q not found", tt.activeHref)
			}
			if !strings.Contains(seg, activeFillClass) {
				t.Errorf("anchor for %q is not rendered as active", tt.activeHref)
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
		t.Fatalf("failed to render: %v", err)
	}
	html := buf.String()

	// Icon-only links must carry an aria-label so screen readers can announce
	// each link by its purpose. Verifying the localized labels also confirms the
	// i18n keys are wired to the correct links.
	// [Ja] アイコンのみのリンクは、各リンクの目的をスクリーンリーダーが読み上げ
	// られるよう aria-label を持つ必要がある。ローカライズ済みラベルを検証する
	// ことで、i18n キーが正しいリンクに紐づいていることも確認する。
	wantLabels := []string{"ホーム", "検索", "新規投稿", "通知", "プロフィール"}
	for _, want := range wantLabels {
		if !strings.Contains(html, `aria-label="`+want+`"`) {
			t.Errorf("NavbarMenu output missing aria-label %q", want)
		}
	}
}

func TestNavbarMenu_ClassName(t *testing.T) {
	t.Parallel()

	navbar := viewmodel.NewNavbar(&model.Profile{Atname: "alice"}, viewmodel.NavbarItemHome)
	html := renderComponent(t, components.NavbarMenu(navbar, "border border-slate-400"))

	if !strings.Contains(html, "border border-slate-400") {
		t.Error("NavbarMenu output missing the appended className")
	}
}

func TestTopNavbar(t *testing.T) {
	t.Parallel()

	navbar := viewmodel.NewNavbar(&model.Profile{Atname: "alice"}, viewmodel.NavbarItemNew)
	html := renderComponent(t, components.TopNavbar(navbar))

	// Logo links to the root and the bar sticks to the top, visible only on lg+.
	// [Ja] ロゴは root へのリンクで、バーは画面上部に固定し lg 以上でのみ表示する。
	checks := []string{`href="/"`, "sticky", "bg-background", "lg:block", "hidden"}
	for _, want := range checks {
		if !strings.Contains(html, want) {
			t.Errorf("TopNavbar output missing %q", want)
		}
	}

	// The menu (with all five links) is embedded in the top navbar.
	// [Ja] 5 項目すべてのメニューがトップ navbar に埋め込まれている。
	if !strings.Contains(html, `href="/@alice"`) {
		t.Error("TopNavbar output missing the navbar menu")
	}
}

func TestBottomNavbar(t *testing.T) {
	t.Parallel()

	navbar := viewmodel.NewNavbar(&model.Profile{Atname: "alice"}, viewmodel.NavbarItemNew)
	html := renderComponent(t, components.BottomNavbar(navbar))

	// The bottom navbar is shown only below lg and carries a border.
	// [Ja] ボトム navbar は lg 未満でのみ表示し、枠線を持つ。
	checks := []string{"lg:hidden", "border border-slate-400", `href="/new"`}
	for _, want := range checks {
		if !strings.Contains(html, want) {
			t.Errorf("BottomNavbar output missing %q", want)
		}
	}
}
