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
		t.Fatalf("描画に失敗: %v", err)
	}
	html := buf.String()

	checks := []string{
		"<!doctype html>", // ドキュメント宣言
		`<html lang="ja"`, // ロケールが反映される
		"テストタイトル | Mewst", // headのタイトル
		"sticky",          // トップnavbar
		`class="flex min-h-[100svh] flex-col pt-safe px-safe"`,
		`<main id="main" tabindex="-1" class="flex-1 pb-safe"`,
		// 固定ラッパーの完全なclass一覧を検証し、モバイル用の透明な
		// ヒット領域がデスクトップのコンテンツ上に残ることを防ぐ。
		`class="fixed bottom-0 left-1/2 z-50 -translate-x-1/2 py-4 px-safe-offset-4 mb-safe lg:hidden"`,
		"<footer",
		`class="mb-[100px] lg:mb-4"`,
		`href="/@alice"`, // navbarメニュー (プロフィールリンク)
		`href="#main"`,   // スキップリンク (WCAG 2.4.1)
		"メインコンテンツへスキップ",                 // スキップリンクのラベル
		`<main id="main" tabindex="-1"`, // mainランドマーク (idはスキップリンクのターゲット、tabindexでプログラム的フォーカスを許可)
		"content-marker",                // 差し込まれたコンテンツ
	}
	for _, want := range checks {
		if !strings.Contains(html, want) {
			t.Errorf("Defaultレイアウトの出力に%qが含まれていない", want)
		}
	}

	// スキップリンクは最初のフォーカス可能要素でなければならないため、DOM上で
	// navbarのリンクより前に現れる必要がある。
	if skip, nav := strings.Index(html, `href="#main"`), strings.Index(html, `href="/@alice"`); skip == -1 || nav == -1 || skip > nav {
		t.Errorf("スキップリンク (位置%d) がnavbarのリンク (位置%d) より前に無い", skip, nav)
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
		t.Fatalf("描画に失敗: %v", err)
	}
	html := buf.String()

	// トップnavbar (lg:block) とボトムnavbar (lg:hidden) の両方が存在し、
	// 同じ5項目メニューが2回描画される。したがって /newリンクはメニュー
	// ごとに1回、合計2回現れる。
	if got := strings.Count(html, `href="/new"`); got != 2 {
		t.Errorf(`href="/new"の件数 = %d、期待値 = 2 (上部とボトムのnavbarメニュー)`, got)
	}
}
