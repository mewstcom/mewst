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

	// 渡した内容がカード本文に収まることを検証できるよう、目印のフォームを
	// childrenとしてカードをレンダリングする。
	child := templ.Raw(`<form data-testid="auth-form"></form>`)
	ctx := templ.WithChildren(context.Background(), child)

	var buf bytes.Buffer
	if err := components.AuthFormCard().Render(ctx, &buf); err != nil {
		t.Fatalf("描画に失敗: %v", err)
	}
	html := buf.String()

	// カードはフォームをカード面に収め (モバイルでは全幅・角なし、md以上では
	// 余白付きの角丸カード)、渡されたフォームをカード本文 (<section>) として描画する。
	for _, want := range []string{"card", "rounded-none", "md:rounded-xl", "<section", `data-testid="auth-form"`} {
		if !strings.Contains(html, want) {
			t.Errorf("AuthFormCardの出力に%qが含まれていない", want)
		}
	}
}
