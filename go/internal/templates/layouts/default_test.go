package layouts_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/templates/layouts"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

func TestDefault(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), "ja")

	data := layouts.DefaultLayoutData{
		Meta:   viewmodel.PageMeta{Title: "テストタイトル | Mewst"},
		Navbar: viewmodel.NewNavbar(&model.Profile{Atname: "alice"}, viewmodel.NavbarItemNew),
	}
	content := templ.Raw(`<p>content-marker</p>`)

	var buf bytes.Buffer
	if err := layouts.Default(data, content).Render(ctx, &buf); err != nil {
		t.Fatalf("failed to render: %v", err)
	}
	html := buf.String()

	checks := []string{
		"<!doctype html>", // ドキュメント宣言
		`<html lang="ja"`, // ロケールが反映される
		"テストタイトル | Mewst", // head のタイトル
		"sticky",          // トップ navbar
		"lg:hidden",       // ボトム navbar
		`href="/@alice"`,  // navbar メニュー (プロフィールリンク)
		`href="#main"`,    // skip link (WCAG 2.4.1). [Ja] スキップリンク (WCAG 2.4.1)
		"メインコンテンツへスキップ",                 // skip link label. [Ja] スキップリンクのラベル
		`<main id="main" tabindex="-1"`, // main landmark (id is the skip link target, tabindex enables programmatic focus). [Ja] main ランドマーク (id はスキップリンクのターゲット、tabindex でプログラム的フォーカスを許可)
		"content-marker",                // 差し込まれたコンテンツ
	}
	for _, want := range checks {
		if !strings.Contains(html, want) {
			t.Errorf("Default layout output missing %q", want)
		}
	}

	// The skip link must be the first focusable element, so it must precede the
	// navbar links in the DOM.
	//
	// [Ja] スキップリンクは最初のフォーカス可能要素でなければならないため、DOM 上で
	// navbar のリンクより前に現れる必要がある。
	if skip, nav := strings.Index(html, `href="#main"`), strings.Index(html, `href="/@alice"`); skip == -1 || nav == -1 || skip > nav {
		t.Errorf("skip link (index %d) must precede navbar links (index %d)", skip, nav)
	}
}

func TestDefault_RendersBothNavbars(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), "ja")

	data := layouts.DefaultLayoutData{
		Meta:   viewmodel.PageMeta{Title: "Mewst"},
		Navbar: viewmodel.NewNavbar(&model.Profile{Atname: "bob"}, viewmodel.NavbarItemHome),
	}

	var buf bytes.Buffer
	if err := layouts.Default(data, templ.Raw("")).Render(ctx, &buf); err != nil {
		t.Fatalf("failed to render: %v", err)
	}
	html := buf.String()

	// Top navbar (lg:block) and bottom navbar (lg:hidden) are both present, so
	// the same five-item menu is rendered twice. The /new link therefore
	// appears once per menu, i.e. twice in total.
	// [Ja] トップ navbar (lg:block) とボトム navbar (lg:hidden) の両方が存在し、
	// 同じ 5 項目メニューが 2 回描画される。したがって /new リンクはメニュー
	// ごとに 1 回、合計 2 回現れる。
	if got := strings.Count(html, `href="/new"`); got != 2 {
		t.Errorf(`href="/new" count = %d, want 2 (top + bottom navbar menus)`, got)
	}
}
