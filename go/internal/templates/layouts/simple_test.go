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
		t.Fatalf("描画に失敗: %v", err)
	}
	html := buf.String()

	checks := []string{
		"<!doctype html>", // ドキュメント宣言。
		`<html lang="ja"`, // ロケールが反映される。
		"テストタイトル | Mewst", // headのタイトル。
		`<body class="min-h-screen flex items-center justify-center p-safe"`,
		// スキップリンク + <main> ランドマーク (WCAG 2.4.1 / semantic-html):
		// スキップリンクのターゲット・ラベル・フォーカス可能なmainラッパー。
		`href="#main"`, // スキップリンクのターゲット。
		"メインコンテンツへスキップ",                 // スキップリンクのラベル。
		`<main id="main" tabindex="-1"`, // mainランドマーク。
		"content-marker",                // 差し込まれたコンテンツ。
	}
	for _, want := range checks {
		if !strings.Contains(html, want) {
			t.Errorf("Simpleレイアウトの出力に%qが含まれていない", want)
		}
	}

	// スキップリンクは最初のフォーカス可能要素でなければならないため、DOM上で
	// メインコンテンツより前に現れる必要がある。
	if skip, main := strings.Index(html, `href="#main"`), strings.Index(html, `<main id="main"`); skip == -1 || main == -1 || skip > main {
		t.Errorf("スキップリンク (位置%d) がmainランドマーク (位置%d) より前に無い", skip, main)
	}
}
