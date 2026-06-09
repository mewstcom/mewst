package components_test

import (
	"strings"
	"testing"

	"github.com/mewstcom/mewst/go/internal/templates/components"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

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
