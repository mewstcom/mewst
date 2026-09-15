package email

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/mewstcom/mewst/go/internal/i18n"
)

// renderComponentはtempl.Componentを文字列にレンダリングするテストヘルパー
func renderComponent(t *testing.T, ctx context.Context, c templ.Component) string {
	t.Helper()
	var buf bytes.Buffer
	if err := c.Render(ctx, &buf); err != nil {
		t.Fatalf("コンポーネントの描画に失敗: %v", err)
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
		t.Fatalf("予期しないエラー: %v", err)
	}

	if len(noopSender.SentEmails) != 1 {
		t.Fatalf("SentEmailsの件数 = %d、期待値 = 1", len(noopSender.SentEmails))
	}

	sent := noopSender.SentEmails[0]

	if sent.To != "test@example.com" {
		t.Errorf("To = %q、期待値 = %q", sent.To, "test@example.com")
	}

	if sent.Subject != "[Mewst] 確認用コード" {
		t.Errorf("Subject = %q、期待値 = %q", sent.Subject, "[Mewst] 確認用コード")
	}

	// templ.ComponentをレンダリングしてHTMLの中身を検証
	htmlStr := renderComponent(t, ctx, sent.HTMLBody)
	if !strings.Contains(htmlStr, "123456") {
		t.Error("HTMLBodyに確認コードが含まれていない")
	}
	if !strings.Contains(htmlStr, "test@example.com") {
		t.Error("HTMLBodyにメールアドレスが含まれていない")
	}

	textStr := renderComponent(t, ctx, sent.TextBody)
	if !strings.Contains(textStr, "123456") {
		t.Error("TextBodyに確認コードが含まれていない")
	}
}

func TestConfirmationSender_Send_English(t *testing.T) {
	t.Parallel()

	noopSender := NewNoopSender()
	sender := NewConfirmationSender(noopSender)

	ctx := i18n.SetLocale(context.Background(), "en")

	err := sender.Send(ctx, "test@example.com", "654321", "en")
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	if len(noopSender.SentEmails) != 1 {
		t.Fatalf("SentEmailsの件数 = %d、期待値 = 1", len(noopSender.SentEmails))
	}

	sent := noopSender.SentEmails[0]

	if sent.Subject != "[Mewst] Confirmation code" {
		t.Errorf("Subject = %q、期待値 = %q", sent.Subject, "[Mewst] Confirmation code")
	}

	htmlStr := renderComponent(t, ctx, sent.HTMLBody)
	if !strings.Contains(htmlStr, "654321") {
		t.Error("HTMLBodyに確認コードが含まれていない")
	}
	if !strings.Contains(htmlStr, `lang="en"`) {
		t.Error("HTMLBodyにlang=enが含まれていない")
	}
}

// TestConfirmationSender_Send_TextBodyKeepsItsParagraphsは、両ロケールの
// text/plain本文の全体を固定する。部分文字列だけを見るテストでは、段落の区切りが
// 空白へつぶれても、"didn't" のアポストロフィがHTML実体参照として出ていても
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
				t.Fatalf("予期しないエラー: %v", err)
			}

			textStr := renderComponent(t, ctx, noopSender.SentEmails[0].TextBody)
			if textStr != tt.want {
				t.Errorf("TextBody = %q、期待値 = %q", textStr, tt.want)
			}
		})
	}
}

// TestConfirmationSender_Send_HTMLBodyEscapesTheEmailは、Text
// テンプレートがメールアドレスをtempl.Rawに渡してよい根拠として挙げている
// エスケープを固定する。mail.ParseAddressはquotedなlocal partを通すため、
// HTMLメタ文字を含むアドレスは新規登録の検証を通り、両方のパートへ到達する。
// それを無害化するのはHTMLパートだけである。
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
				t.Fatalf("予期しないエラー: %v", err)
			}

			htmlStr := renderComponent(t, ctx, noopSender.SentEmails[0].HTMLBody)

			if strings.Contains(htmlStr, "<img") {
				t.Errorf("HTMLBodyにエスケープされていないマークアップが含まれている: %q", htmlStr)
			}
			if !strings.Contains(htmlStr, "&lt;img src=x onerror=alert(1)&gt;") {
				t.Errorf("HTMLBodyにエスケープしたアドレスが含まれていない: %q", htmlStr)
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
		t.Fatalf("予期しないエラー: %v", err)
	}

	sent := noopSender.SentEmails[0]

	// 未知のロケールは日本語のテンプレートにフォールバック
	ctx := i18n.SetLocale(context.Background(), "ja")
	htmlStr := renderComponent(t, ctx, sent.HTMLBody)
	if !strings.Contains(htmlStr, "111111") {
		t.Error("HTMLBodyに確認コードが含まれていない")
	}
	if !strings.Contains(htmlStr, `lang="ja"`) {
		t.Error("HTMLBodyにlang=jaが含まれていない")
	}
}
