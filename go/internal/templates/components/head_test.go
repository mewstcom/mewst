package components_test

import (
	"strings"
	"testing"

	"github.com/mewstcom/mewst/go/internal/templates/components"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

func TestHead_ViewportAllowsZoom(t *testing.T) {
	t.Parallel()

	meta := viewmodel.PageMeta{
		Title:        "Test",
		Description:  "Test description",
		AssetVersion: "test",
	}
	html := renderComponent(t, components.Head(meta))

	// Pinch-zoom must stay enabled (WCAG 1.4.4): the viewport must not pin the
	// maximum scale or disable user scaling. iOS Safari's focus auto-zoom is
	// avoided by 16px form inputs, not by locking the viewport (see head.templ).
	//
	// [Ja] ピンチズームは有効なまま保つ必要がある (WCAG 1.4.4)。viewport で
	// 最大スケールを固定したりユーザースケーリングを無効化したりしてはならない。
	// iOS Safari のフォーカス時自動ズームは viewport の固定ではなく 16px の
	// フォーム入力で回避している (head.templ を参照)。
	required := []string{"width=device-width", "initial-scale=1"}
	for _, want := range required {
		if !strings.Contains(html, want) {
			t.Errorf("Head viewport must contain %q, got: %q", want, html)
		}
	}
	forbidden := []string{"maximum-scale", "user-scalable"}
	for _, want := range forbidden {
		if strings.Contains(html, want) {
			t.Errorf("Head viewport must not contain %q (it would disable pinch-zoom)", want)
		}
	}
}

func TestHead_NoDarkModeScript(t *testing.T) {
	t.Parallel()

	meta := viewmodel.PageMeta{
		Title:        "Test",
		Description:  "Test description",
		AssetVersion: "test",
	}
	html := renderComponent(t, components.Head(meta))

	// Dark mode is disabled during the Rails-to-Go migration, so the rendered
	// head must not contain the detection script that adds the `.dark` class
	// from the OS color-scheme preference (see head.templ for why).
	//
	// [Ja] 移行期はダークモードを無効化しているため、描画された head には OS の
	// カラースキーム設定から `.dark` クラスを付与する検出スクリプトが含まれては
	// ならない (理由は head.templ を参照)。
	forbidden := []string{
		"prefers-color-scheme: dark",
		`classList.add("dark")`,
	}
	for _, want := range forbidden {
		if strings.Contains(html, want) {
			t.Errorf("Head output must not contain dark mode detection %q", want)
		}
	}
}
