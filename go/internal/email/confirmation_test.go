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

// TestConfirmationSender_Send_TextBodyKeepsItsParagraphs pins the whole
// text/plain body of both locales. Checking substrings alone let the paragraph
// breaks collapse into spaces unnoticed, and let the apostrophe in "didn't" go
// out as an HTML entity.
//
// [Ja] TestConfirmationSender_Send_TextBodyKeepsItsParagraphs は、両ロケールの
// text/plain 本文の全体を固定する。部分文字列だけを見るテストでは、段落の区切りが
// 空白へつぶれても、"didn't" のアポストロフィが HTML 実体参照として出ていても
// 気付けなかった。
func TestConfirmationSender_Send_TextBodyKeepsItsParagraphs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		locale string
		want   string
	}{
		{
			locale: "ja",
			want: "test@example.com さん、こんにちは。\n\n" +
				"確認用コードは下記になります。\n\n" +
				"123456\n\n" +
				"確認用コードの有効期間は15分です。\n\n" +
				"もしこのメールに心当たりが無い場合は無視してください。\n\n" +
				"-- \nMewst\nhttps://mewst.com\n",
		},
		{
			locale: "en",
			want: "Hello test@example.com,\n\n" +
				"Your confirmation code is below.\n\n" +
				"123456\n\n" +
				"This confirmation code will expire in 15 minutes.\n\n" +
				"If you didn't request this email, please ignore it.\n\n" +
				"-- \nMewst\nhttps://mewst.com\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.locale, func(t *testing.T) {
			t.Parallel()

			noopSender := NewNoopSender()
			sender := NewConfirmationSender(noopSender)

			ctx := i18n.SetLocale(context.Background(), tt.locale)

			err := sender.Send(ctx, "test@example.com", "123456", tt.locale)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			textStr := renderComponent(t, ctx, noopSender.SentEmails[0].TextBody)
			if textStr != tt.want {
				t.Errorf("TextBody = %q, want %q", textStr, tt.want)
			}
		})
	}
}

// TestConfirmationSender_Send_HTMLBodyEscapesTheEmail pins the escaping that
// the text templates name as the reason they may hand the address to
// templ.Raw. mail.ParseAddress accepts a quoted local part, so an address
// carrying HTML metacharacters passes sign-up validation and reaches both
// parts; only the HTML part must neutralize it.
//
// [Ja] TestConfirmationSender_Send_HTMLBodyEscapesTheEmail は、Text
// テンプレートがメールアドレスを templ.Raw に渡してよい根拠として挙げている
// エスケープを固定する。mail.ParseAddress は quoted な local part を通すため、
// HTML メタ文字を含むアドレスは新規登録の検証を通り、両方のパートへ到達する。
// それを無害化するのは HTML パートだけである。
func TestConfirmationSender_Send_HTMLBodyEscapesTheEmail(t *testing.T) {
	t.Parallel()

	const email = `"<img src=x onerror=alert(1)>"@example.com`

	for _, locale := range []string{"ja", "en"} {
		t.Run(locale, func(t *testing.T) {
			t.Parallel()

			noopSender := NewNoopSender()
			sender := NewConfirmationSender(noopSender)

			ctx := i18n.SetLocale(context.Background(), locale)

			err := sender.Send(ctx, email, "123456", locale)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			htmlStr := renderComponent(t, ctx, noopSender.SentEmails[0].HTMLBody)

			if strings.Contains(htmlStr, "<img") {
				t.Errorf("HTMLBody contains unescaped markup: %q", htmlStr)
			}
			if !strings.Contains(htmlStr, "&lt;img src=x onerror=alert(1)&gt;") {
				t.Errorf("HTMLBody does not contain the escaped address: %q", htmlStr)
			}
		})
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
