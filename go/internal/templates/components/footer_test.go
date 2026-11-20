package components_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/templates/components"
)

func TestBasicFooter(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), "ja")

	var buf bytes.Buffer
	if err := components.BasicFooter().Render(ctx, &buf); err != nil {
		t.Fatalf("描画に失敗: %v", err)
	}
	html := buf.String()

	for _, want := range []string{
		"<footer",
		`aria-label="Mewst"`,
		`href="/"`,
		"font-['Itim']",
		"Mewst",
		`href="/community"`, `<span lang="en">Community</span>`,
		`href="https://wikino.app/s/mewst/topics/1"`, `<span lang="en">Help</span>`,
		`href="/terms"`, `<span lang="en">Terms</span>`,
		`href="/privacy"`, `<span lang="en">Privacy</span>`,
		`target="_blank"`,
		`rel="nofollow noopener"`,
		"sr-only",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("BasicFooterの出力に%qが含まれていない", want)
		}
	}

	// 4つのフッターリンクはそれぞれ新規タブで開くため、target/relはリンクごとに
	// 1回ずつ現れる。ワードマークのリンクは同一タブで開くため付かない。
	if got := strings.Count(html, `target="_blank"`); got != 4 {
		t.Errorf(`target="_blank"の件数 = %d、期待値 = 4`, got)
	}
	if got := strings.Count(html, `rel="nofollow noopener"`); got != 4 {
		t.Errorf(`rel="nofollow noopener"の件数 = %d、期待値 = 4`, got)
	}
	if got := strings.Count(html, `lang="en"`); got != 4 {
		t.Errorf(`lang="en"の件数 = %d、期待値 = 4`, got)
	}
	if got := strings.Count(html, "inline-flex min-h-6 items-center"); got != 4 {
		t.Errorf("最小タッチターゲットのクラスの件数 = %d、期待値 = 4", got)
	}
}

func TestBasicFooter_NewTabHintLocalized(t *testing.T) {
	t.Parallel()

	// 新規タブのヒントは現在のロケールに従い、表示ラベルは英語固定とする。
	tests := []struct {
		name   string
		locale string
		want   string
	}{
		{name: "日本語", locale: "ja", want: "新しいタブで開く"},
		{name: "英語", locale: "en", want: "Opens in new tab"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := i18n.SetLocale(context.Background(), tt.locale)

			var buf bytes.Buffer
			if err := components.BasicFooter().Render(ctx, &buf); err != nil {
				t.Fatalf("描画に失敗: %v", err)
			}

			if !strings.Contains(buf.String(), tt.want) {
				t.Errorf("BasicFooter (%s) の出力に新しいタブで開く旨の%qが含まれていない", tt.locale, tt.want)
			}
		})
	}
}
