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
		"<!doctype html>",       // ドキュメント宣言
		`<html lang="ja"`,       // ロケールが反映される
		"テストタイトル | Mewst",       // head のタイトル
		"sticky",                // トップ navbar
		"lg:hidden",             // ボトム navbar
		`href="/@alice"`,        // navbar メニュー (プロフィールリンク)
		`<main class="flex-1">`, // メインコンテンツ領域
		"content-marker",        // 差し込まれたコンテンツ
	}
	for _, want := range checks {
		if !strings.Contains(html, want) {
			t.Errorf("Default layout output missing %q", want)
		}
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
