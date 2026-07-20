package components_test

import (
	"strings"
	"testing"

	"github.com/mewstcom/mewst/go/internal/templates/components"
)

func TestLogoLink(t *testing.T) {
	t.Parallel()

	html := renderComponent(t, components.LogoLink())

	// The link points to the site root and carries the Mewst brand name, so even
	// an icon-only caller exposes an accessible name.
	//
	// [Ja] リンクはサイトルートを指し、Mewst のブランド名を持つため、アイコンのみの
	// 呼び出し元でもアクセシブルな名前が露出する。
	for _, want := range []string{`href="/"`, `aria-label="Mewst"`, "inline-block"} {
		if !strings.Contains(html, want) {
			t.Errorf("LogoLink output missing %q", want)
		}
	}
}

func TestLogoTile(t *testing.T) {
	t.Parallel()

	html := renderComponent(t, components.LogoTile())

	// The tile links home (via LogoLink) and frames the brand logo glyph on a
	// primary-colored surface.
	//
	// [Ja] タイルは (LogoLink を介して) ホームへリンクし、primary 色の面の上に
	// ブランドロゴのグリフを収める。
	for _, want := range []string{`href="/"`, `aria-label="Mewst"`, "bg-primary", "rounded-xl", "fill-black", `fill="currentColor"`, "<svg"} {
		if !strings.Contains(html, want) {
			t.Errorf("LogoTile output missing %q", want)
		}
	}

	// The glyph inherits its fill from the root SVG so the LogoTile fill utility
	// controls the rendered logo color.
	//
	// [Ja] LogoTile の fill ユーティリティが描画色を制御できるよう、グリフは
	// ルート SVG から fill を継承する。
	if strings.Contains(html, `<path fill=`) {
		t.Error("LogoTile path must not define its own fill")
	}
}
