package layouts_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/templates/layouts"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

func TestSimple(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), "ja")

	data := layouts.SimpleLayoutData{
		Meta: viewmodel.PageMeta{Title: "テストタイトル | Mewst"},
	}
	content := templ.Raw(`<p>content-marker</p>`)

	var buf bytes.Buffer
	if err := layouts.Simple(data, content).Render(ctx, &buf); err != nil {
		t.Fatalf("failed to render: %v", err)
	}
	html := buf.String()

	checks := []string{
		"<!doctype html>", // ドキュメント宣言。[Ja] document declaration
		`<html lang="ja"`, // ロケールが反映される。[Ja] locale is applied
		"テストタイトル | Mewst", // head のタイトル。[Ja] head title
		`<body class="min-h-screen flex items-center justify-center p-safe"`,
		// Skip link + <main> landmark (WCAG 2.4.1 / semantic-html): the skip link
		// target, its label, and the focusable main wrapper.
		//
		// [Ja] スキップリンク + <main> ランドマーク (WCAG 2.4.1 / semantic-html):
		// スキップリンクのターゲット・ラベル・フォーカス可能な main ラッパー。
		`href="#main"`, // スキップリンクのターゲット。[Ja] skip link target
		"メインコンテンツへスキップ",                 // スキップリンクのラベル。[Ja] skip link label
		`<main id="main" tabindex="-1"`, // main ランドマーク。[Ja] main landmark
		"content-marker",                // 差し込まれたコンテンツ。[Ja] injected content
	}
	for _, want := range checks {
		if !strings.Contains(html, want) {
			t.Errorf("Simple layout output missing %q", want)
		}
	}

	// The skip link must be the first focusable element, so it must precede the
	// main content in the DOM.
	//
	// [Ja] スキップリンクは最初のフォーカス可能要素でなければならないため、DOM 上で
	// メインコンテンツより前に現れる必要がある。
	if skip, main := strings.Index(html, `href="#main"`), strings.Index(html, `<main id="main"`); skip == -1 || main == -1 || skip > main {
		t.Errorf("skip link (index %d) must precede the main landmark (index %d)", skip, main)
	}
}
