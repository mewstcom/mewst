package layouts_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/templates"
	"github.com/mewstcom/mewst/go/internal/templates/layouts"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

func TestCompose(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), "ja")

	data := layouts.ComposeLayoutData{
		Meta:     viewmodel.PageMeta{Title: "テストタイトル | Mewst"},
		BackHref: templates.HomePath(),
	}
	content := templ.Raw(`<p>content-marker</p>`)

	var buf bytes.Buffer
	if err := layouts.Compose(data, content).Render(ctx, &buf); err != nil {
		t.Fatalf("failed to render: %v", err)
	}
	html := buf.String()

	checks := []string{
		"<!doctype html>", // ドキュメント宣言。[Ja] document declaration
		`<html lang="ja"`, // ロケールが反映される。[Ja] locale is applied
		"テストタイトル | Mewst", // head のタイトル。[Ja] head title
		// Back affordance pinned to the top-left corner as layout chrome (BackLink
		// component): the JS hook, the /home fallback href, and the visible label.
		// [Ja] レイアウト共通 chrome として左上端に固定する戻る導線 (BackLink コンポーネント):
		// JS フック・/home フォールバック href・可視ラベル。
		"fixed",          // 左上端への固定配置。[Ja] top-left corner pinning
		"data-back-link", // 戻るリンクの JS フック。[Ja] back-link JS hook
		`href="/home"`,   // 戻る導線のフォールバック先。[Ja] back affordance fallback
		"戻る",             // 戻るリンクのラベル (back_link)。[Ja] back link label
		"content-marker", // 差し込まれたコンテンツ。[Ja] injected content
	}
	for _, want := range checks {
		if !strings.Contains(html, want) {
			t.Errorf("Compose layout output missing %q", want)
		}
	}

	// Compose must omit the navbar so it stays a focused, single-task screen; a
	// navbar-only link (e.g. search) must not appear.
	// [Ja] Compose は集中作成画面を保つため navbar を出さない。navbar 専用リンク
	// (検索など) が現れてはならない。
	if strings.Contains(html, `href="/search"`) {
		t.Error("navbar 無しの Compose に navbar の検索リンクが含まれています")
	}
}
