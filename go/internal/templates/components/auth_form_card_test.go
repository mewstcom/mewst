package components_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/mewstcom/mewst/go/internal/templates/components"
)

func TestAuthFormCard(t *testing.T) {
	t.Parallel()

	// Render the card with a marker form as its children so we can assert the
	// supplied content lands inside the card body.
	//
	// [Ja] 渡した内容がカード本文に収まることを検証できるよう、目印のフォームを
	// children としてカードをレンダリングする。
	child := templ.Raw(`<form data-testid="auth-form"></form>`)
	ctx := templ.WithChildren(context.Background(), child)

	var buf bytes.Buffer
	if err := components.AuthFormCard().Render(ctx, &buf); err != nil {
		t.Fatalf("failed to render: %v", err)
	}
	html := buf.String()

	// The card frames the form on a card surface—full-bleed with square corners
	// on mobile, an inset rounded card from md up—and renders the supplied form
	// as its card body (<section>).
	//
	// [Ja] カードはフォームをカード面に収め (モバイルでは全幅・角なし、md 以上では
	// 余白付きの角丸カード)、渡されたフォームをカード本文 (<section>) として描画する。
	for _, want := range []string{"card", "rounded-none", "md:rounded-xl", "<section", `data-testid="auth-form"`} {
		if !strings.Contains(html, want) {
			t.Errorf("AuthFormCard output missing %q", want)
		}
	}
}
