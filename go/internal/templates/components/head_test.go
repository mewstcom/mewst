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

	// ピンチズームは有効なまま保つ必要がある (WCAG 1.4.4)。viewportで
	// 最大スケールを固定したりユーザースケーリングを無効化したりしてはならない。
	// iOS Safariのフォーカス時自動ズームはviewportの固定ではなく16pxの
	// フォーム入力で回避している (head.templを参照)。
	required := []string{"width=device-width", "initial-scale=1", "viewport-fit=cover"}
	for _, want := range required {
		if !strings.Contains(html, want) {
			t.Errorf("Headのviewportに%qが含まれていない: 実測値 = %q", want, html)
		}
	}
	forbidden := []string{"maximum-scale", "user-scalable"}
	for _, want := range forbidden {
		if strings.Contains(html, want) {
			t.Errorf("Headのviewportに%qが含まれている (ピンチズームが無効になる)", want)
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

	// 移行期はダークモードを無効化しているため、描画されたheadにはOSの
	// カラースキーム設定から `.dark` クラスを付与する検出スクリプトが含まれては
	// ならない (理由はhead.templを参照)。
	forbidden := []string{
		"prefers-color-scheme: dark",
		`classList.add("dark")`,
	}
	for _, want := range forbidden {
		if strings.Contains(html, want) {
			t.Errorf("Headの出力にダークモード検出の%qが含まれている", want)
		}
	}
}
