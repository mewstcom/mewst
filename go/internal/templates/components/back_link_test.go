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

	if strings.Contains(html, "btn-outline") {
		t.Error("BackLink should not render btn-outline")
	}
}
