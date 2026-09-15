package components_test

import (
	"strings"
	"testing"

	"github.com/mewstcom/mewst/go/internal/templates/components"
)

func TestLogoLink(t *testing.T) {
	t.Parallel()

	html := renderComponent(t, components.LogoLink())

	// リンクはサイトルートを指し、Mewstのブランド名を持つため、アイコンのみの
	// 呼び出し元でもアクセシブルな名前が露出する。
	for _, want := range []string{`href="/"`, `aria-label="Mewst"`, "inline-block"} {
		if !strings.Contains(html, want) {
			t.Errorf("LogoLinkの出力に%qが含まれていない", want)
		}
	}
}

func TestLogoTile(t *testing.T) {
	t.Parallel()

	html := renderComponent(t, components.LogoTile())

	// タイルは (LogoLinkを介して) ホームへリンクし、primary色の面の上に
	// ブランドロゴのグリフを収める。
	for _, want := range []string{`href="/"`, `aria-label="Mewst"`, "bg-primary", "rounded-xl", "fill-black", `fill="currentColor"`, "<svg"} {
		if !strings.Contains(html, want) {
			t.Errorf("LogoTileの出力に%qが含まれていない", want)
		}
	}

	// LogoTileのfillユーティリティが描画色を制御できるよう、グリフは
	// ルートSVGからfillを継承する。
	if strings.Contains(html, `<path fill=`) {
		t.Error("LogoTileのpathが独自のfillを持っている")
	}
}
