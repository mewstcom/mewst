package email

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/mewstcom/mewst/go/internal/i18n"
)

// renderComponent はtempl.Componentを文字列にレンダリングするテストヘルパー
func renderComponent(t *testing.T, ctx context.Context, c templ.Component) string {
	t.Helper()
	var buf bytes.Buffer
	if err := c.Render(ctx, &buf); err != nil {
		t.Fatalf("failed to render component: %v", err)
	}
	return buf.String()
}

func TestConfirmationSender_Send_Japanese(t *testing.T) {
	t.Parallel()

	noopSender := NewNoopSender()
	sender := NewConfirmationSender(noopSender)

	ctx := i18n.SetLocale(context.Background(), "ja")

	err := sender.Send(ctx, "test@example.com", "123456", "ja")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(noopSender.SentEmails) != 1 {
		t.Fatalf("SentEmails count = %d, want 1", len(noopSender.SentEmails))
	}

	sent := noopSender.SentEmails[0]

	if sent.To != "test@example.com" {
		t.Errorf("To = %q, want %q", sent.To, "test@example.com")
	}

	if sent.Subject != "[Mewst] 確認用コード" {
		t.Errorf("Subject = %q, want %q", sent.Subject, "[Mewst] 確認用コード")
	}

	// templ.ComponentをレンダリングしてHTMLの中身を検証
	htmlStr := renderComponent(t, ctx, sent.HTMLBody)
	if !strings.Contains(htmlStr, "123456") {
		t.Error("HTMLBody does not contain confirmation code")
	}
	if !strings.Contains(htmlStr, "test@example.com") {
		t.Error("HTMLBody does not contain email address")
	}

	textStr := renderComponent(t, ctx, sent.TextBody)
	if !strings.Contains(textStr, "123456") {
		t.Error("TextBody does not contain confirmation code")
	}
}

func TestConfirmationSender_Send_English(t *testing.T) {
	t.Parallel()

	noopSender := NewNoopSender()
	sender := NewConfirmationSender(noopSender)

	ctx := i18n.SetLocale(context.Background(), "en")

	err := sender.Send(ctx, "test@example.com", "654321", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(noopSender.SentEmails) != 1 {
		t.Fatalf("SentEmails count = %d, want 1", len(noopSender.SentEmails))
	}

	sent := noopSender.SentEmails[0]

	if sent.Subject != "[Mewst] Confirmation code" {
		t.Errorf("Subject = %q, want %q", sent.Subject, "[Mewst] Confirmation code")
	}

	htmlStr := renderComponent(t, ctx, sent.HTMLBody)
	if !strings.Contains(htmlStr, "654321") {
		t.Error("HTMLBody does not contain confirmation code")
	}
	if !strings.Contains(htmlStr, `lang="en"`) {
		t.Error("HTMLBody does not contain lang=en")
	}
}

func TestConfirmationSender_Send_UnknownLocale_FallsBackToJapanese(t *testing.T) {
	t.Parallel()

	noopSender := NewNoopSender()
	sender := NewConfirmationSender(noopSender)

	err := sender.Send(context.Background(), "test@example.com", "111111", "unknown")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sent := noopSender.SentEmails[0]

	// 未知のロケールは日本語のテンプレートにフォールバック
	ctx := i18n.SetLocale(context.Background(), "ja")
	htmlStr := renderComponent(t, ctx, sent.HTMLBody)
	if !strings.Contains(htmlStr, "111111") {
		t.Error("HTMLBody does not contain confirmation code")
	}
	if !strings.Contains(htmlStr, `lang="ja"`) {
		t.Error("HTMLBody does not contain lang=ja")
	}
}
