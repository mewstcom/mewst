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

func TestCentered(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), "ja")

	data := layouts.CenteredLayoutData{
		Meta:   viewmodel.PageMeta{Title: "テストタイトル | Mewst"},
		Navbar: viewmodel.NewNavbar(&model.Profile{Atname: "alice"}, viewmodel.NavbarItemNew),
	}
	content := templ.Raw(`<p>content-marker</p>`)

	var buf bytes.Buffer
	if err := layouts.Centered(data, content).Render(ctx, &buf); err != nil {
		t.Fatalf("failed to render: %v", err)
	}
	html := buf.String()

	// The centered layout must render the document metadata, both responsive
	// navbars, the skip link and main landmark, and the injected page content.
	// Checking the fixed wrapper's complete class list also prevents its mobile
	// hit area from remaining over desktop content.
	//
	// [Ja] 中央寄せレイアウトがドキュメントメタデータ、レスポンシブな両 navbar、
	// スキップリンクと main ランドマーク、差し込んだページ内容を描画することを検証する。
	// 固定ラッパーの完全なクラス一覧も検証し、モバイル用のヒット領域が PC の
	// コンテンツ上に残ることを防ぐ。
	checks := []string{
		"<!doctype html>",
		`<html lang="ja"`,
		"テストタイトル | Mewst",
		"sticky",
		`class="flex min-h-[100svh] flex-col pt-safe px-safe"`,
		`<main id="main" tabindex="-1" class="flex-1 grid place-items-center pb-safe-offset-24 lg:pb-safe-offset-8"`,
		`class="fixed bottom-0 left-1/2 z-50 -translate-x-1/2 py-4 px-safe-offset-4 mb-safe lg:hidden"`,
		`href="/@alice"`,
		`href="#main"`,
		"メインコンテンツへスキップ",
		`<main id="main" tabindex="-1"`,
		"content-marker",
	}
	for _, want := range checks {
		if !strings.Contains(html, want) {
			t.Errorf("Centered layout output missing %q", want)
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

func TestCentered_RendersBothNavbars(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), "ja")

	data := layouts.CenteredLayoutData{
		Meta:   viewmodel.PageMeta{Title: "Mewst"},
		Navbar: viewmodel.NewNavbar(&model.Profile{Atname: "bob"}, viewmodel.NavbarItemNew),
	}

	var buf bytes.Buffer
	if err := layouts.Centered(data, templ.Raw("")).Render(ctx, &buf); err != nil {
		t.Fatalf("failed to render: %v", err)
	}
	html := buf.String()

	// Top navbar (lg:block) and bottom navbar (lg:hidden) are both present, so the
	// same five-item menu is rendered twice. The /new link therefore appears once
	// per menu, i.e. twice in total.
	//
	// [Ja] トップ navbar (lg:block) とボトム navbar (lg:hidden) の両方が存在し、
	// 同じ 5 項目メニューが 2 回描画される。したがって /new リンクはメニューごとに
	// 1 回、合計 2 回現れる。
	if got := strings.Count(html, `href="/new"`); got != 2 {
		t.Errorf(`href="/new" count = %d, want 2 (top + bottom navbar menus)`, got)
	}

	// /new is the active menu item on this layout, so exactly one item per menu
	// (two in total) renders the active filled-icon fill override and exposes
	// aria-current="page".
	//
	// [Ja] このレイアウトでは /new がアクティブなメニュー項目のため、メニューごとに
	// ちょうど 1 項目 (合計 2 項目) がアクティブの塗りつぶしアイコンの fill 上書きを描画し、
	// aria-current="page" を公開する。
	if got := strings.Count(html, "[&_.content]:fill-foreground"); got != 2 {
		t.Errorf("active fill class count = %d, want 2 (new active in top + bottom navbar menus)", got)
	}
	if got := strings.Count(html, `aria-current="page"`); got != 2 {
		t.Errorf(`aria-current="page" count = %d, want 2 (new active in top + bottom navbar menus)`, got)
	}
}
