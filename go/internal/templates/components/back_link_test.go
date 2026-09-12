package components_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/templates"
	"github.com/mewstcom/mewst/go/internal/templates/components"
)

func TestBackLink(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), "ja")

	var buf bytes.Buffer
	if err := components.BackLink(templates.HomePath()).Render(ctx, &buf); err != nil {
		t.Fatalf("failed to render: %v", err)
	}
	html := buf.String()
	const backArrowIconPathFragment = "M232,200a8"

	for _, want := range []string{
		"<a ",
		"data-back-link",
		`href="/home"`,
		"link-bare-muted-foreground",
		"text-sm",
		"focus-visible:outline-2",
		"focus-visible:outline-offset-2",
		"focus-visible:outline-foreground",
		backArrowIconPathFragment,
		"戻る",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("BackLink output missing %q", want)
		}
	}

	// BackLink is a bare muted link, so it must carry no button chrome. Basecoat
	// 1.0 composes buttons as the `btn` root class plus `data-*` modifiers, so
	// assert on both instead of the removed `btn-outline` alias.
	//
	// [Ja] BackLink は枠なしの muted なリンクであり、ボタン風の装飾を持たない。
	// Basecoat 1.0 のボタンは root クラス `btn` と `data-*` 修飾子の組み合わせで
	// 表現するため、削除された `btn-outline` エイリアスではなく両方をアサートする。
	for _, unwanted := range []string{`class="btn`, `data-variant="outline"`} {
		if strings.Contains(html, unwanted) {
			t.Errorf("BackLink should not render button chrome %q", unwanted)
		}
	}
}
