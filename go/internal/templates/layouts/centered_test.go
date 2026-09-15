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
		t.Fatalf("描画に失敗: %v", err)
	}
	html := buf.String()

	// 中央寄せレイアウトがドキュメントメタデータ、レスポンシブな両navbar、
	// スキップリンクとmainランドマーク、差し込んだページ内容を描画することを検証する。
	// 固定ラッパーの完全なクラス一覧も検証し、モバイル用のヒット領域がPCの
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
			t.Errorf("Centeredレイアウトの出力に%qが含まれていない", want)
		}
	}

	// スキップリンクは最初のフォーカス可能要素でなければならないため、DOM上で
	// navbarのリンクより前に現れる必要がある。
	if skip, nav := strings.Index(html, `href="#main"`), strings.Index(html, `href="/@alice"`); skip == -1 || nav == -1 || skip > nav {
		t.Errorf("スキップリンク (位置%d) がnavbarのリンク (位置%d) より前に無い", skip, nav)
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
		t.Fatalf("描画に失敗: %v", err)
	}
	html := buf.String()

	// トップnavbar (lg:block) とボトムnavbar (lg:hidden) の両方が存在し、
	// 同じ5項目メニューが2回描画される。したがって /newリンクはメニューごとに
	// 1回、合計2回現れる。
	if got := strings.Count(html, `href="/new"`); got != 2 {
		t.Errorf(`href="/new"の件数 = %d、期待値 = 2 (上部とボトムのnavbarメニュー)`, got)
	}

	// このレイアウトでは /newがアクティブなメニュー項目のため、メニューごとに
	// ちょうど1項目 (合計2項目) がアクティブの塗りつぶしアイコンのfill上書きを描画し、
	// aria-current="page" を公開する。
	if got := strings.Count(html, "[&_.content]:fill-foreground"); got != 2 {
		t.Errorf("アクティブの塗りクラスの件数 = %d、期待値 = 2 (上部とボトムのnavbarメニューで新規投稿がアクティブ)", got)
	}
	if got := strings.Count(html, `aria-current="page"`); got != 2 {
		t.Errorf(`aria-current="page"の件数 = %d、期待値 = 2 (上部とボトムのnavbarメニューで新規投稿がアクティブ)`, got)
	}
}
