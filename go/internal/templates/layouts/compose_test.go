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
		"<!doctype html>",    // ドキュメント宣言
		`<html lang="ja"`,    // ロケールが反映される
		"テストタイトル | Mewst",    // head のタイトル
		"items-center",       // コンテンツを中央寄せする
		"justify-center",     // コンテンツを中央寄せする
		"data-back-link",     // 戻るリンクの JS フック
		`href="/home"`,       // 戻る導線のフォールバック先 (BackHref)
		"戻る",                 // 戻るリンクのラベル (compose_back)
		"fixed left-4 top-4", // 戻る導線をウィンドウ左上端に固定する
		"content-marker",     // 差し込まれたコンテンツ
	}
	for _, want := range checks {
		if !strings.Contains(html, want) {
			t.Errorf("Compose layout output missing %q", want)
		}
	}

	// The compose layout is navbar-less by design, so the shared navbar markup
	// (sticky top navbar, lg:hidden bottom navbar) must not appear.
	// [Ja] compose レイアウトは設計上 navbar を持たないため、共通 navbar のマークアップ
	// (sticky なトップ navbar・lg:hidden のボトム navbar) は出力されない。
	for _, navbarMarker := range []string{"sticky", "lg:hidden"} {
		if strings.Contains(html, navbarMarker) {
			t.Errorf("Compose layout unexpectedly rendered navbar marker %q", navbarMarker)
		}
	}
}
