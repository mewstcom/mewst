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
		t.Fatalf("failed to render: %v", err)
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
			t.Errorf("BasicFooter output missing %q", want)
		}
	}

	// Each of the four footer links opens in a new tab, so target/rel appear
	// once per link. The wordmark link is same-tab and must not carry them.
	//
	// [Ja] 4 つのフッターリンクはそれぞれ新規タブで開くため、target/rel はリンクごとに
	// 1 回ずつ現れる。ワードマークのリンクは同一タブで開くため付かない。
	if got := strings.Count(html, `target="_blank"`); got != 4 {
		t.Errorf(`target="_blank" count = %d, want 4`, got)
	}
	if got := strings.Count(html, `rel="nofollow noopener"`); got != 4 {
		t.Errorf(`rel="nofollow noopener" count = %d, want 4`, got)
	}
	if got := strings.Count(html, `lang="en"`); got != 4 {
		t.Errorf(`lang="en" count = %d, want 4`, got)
	}
	if got := strings.Count(html, "inline-flex min-h-6 items-center"); got != 4 {
		t.Errorf("minimum touch target class count = %d, want 4", got)
	}
}

func TestBasicFooter_NewTabHintLocalized(t *testing.T) {
	t.Parallel()

	// The new-tab hint follows the current locale while the visible labels stay
	// fixed English strings.
	//
	// [Ja] 新規タブのヒントは現在のロケールに従い、表示ラベルは英語固定とする。
	tests := []struct {
		name   string
		locale string
		want   string
	}{
		{name: "Japanese", locale: "ja", want: "新しいタブで開く"},
		{name: "English", locale: "en", want: "Opens in new tab"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := i18n.SetLocale(context.Background(), tt.locale)

			var buf bytes.Buffer
			if err := components.BasicFooter().Render(ctx, &buf); err != nil {
				t.Fatalf("failed to render: %v", err)
			}

			if !strings.Contains(buf.String(), tt.want) {
				t.Errorf("BasicFooter (%s) output missing new-tab hint %q", tt.locale, tt.want)
			}
		})
	}
}
