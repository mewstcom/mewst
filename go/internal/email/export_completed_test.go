package email

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/mewstcom/mewst/go/internal/i18n"
)

// errSenderFailed stands in for a delivery failure of the underlying Sender.
//
// [Ja] errSenderFailed は下位の Sender の配信失敗を代弁する。
var errSenderFailed = errors.New("送信に失敗しました")

// failingSender fails every send, so that the caller's error handling can be
// exercised without reaching a mail provider.
//
// [Ja] failingSender はすべての送信に失敗する。メールのプロバイダーへ到達せずに
// 呼び出し元のエラー処理を試すため。
type failingSender struct{}

func (s *failingSender) Send(_ context.Context, _ SendInput) error {
	return errSenderFailed
}

func TestExportCompletedSender_Send_Japanese(t *testing.T) {
	t.Parallel()

	const exportURL = "https://mewst.com/settings/export"

	noopSender := NewNoopSender()
	sender := NewExportCompletedSender(noopSender)

	ctx := i18n.SetLocale(context.Background(), "ja")

	err := sender.Send(ctx, "test@example.com", exportURL, "ja", "01J000000000000000000EXPRT")
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

	if sent.Subject != "[Mewst] エクスポートの準備ができました" {
		t.Errorf("Subject = %q, want %q", sent.Subject, "[Mewst] エクスポートの準備ができました")
	}

	// The key is derived from the export, so that a job retried after a lost
	// response delivers the same notification once.
	//
	// [Ja] キーはエクスポートから導出する。応答を失った後に再試行されたジョブが
	// 同じ通知を 1 度だけ配信するため。
	if sent.IdempotencyKey != "export-completed/01J000000000000000000EXPRT" {
		t.Errorf("IdempotencyKey = %q, want %q", sent.IdempotencyKey, "export-completed/01J000000000000000000EXPRT")
	}

	htmlStr := renderComponent(t, ctx, sent.HTMLBody)
	if !strings.Contains(htmlStr, exportURL) {
		t.Error("HTMLBody does not contain the export URL")
	}
	if !strings.Contains(htmlStr, ">エクスポート画面を開く</a>") {
		t.Error("HTMLBody does not contain descriptive export link text")
	}
	if !strings.Contains(htmlStr, ">Mewst 公式サイト</a>") {
		t.Error("HTMLBody does not contain descriptive website link text")
	}
	if !strings.Contains(htmlStr, "新しいエクスポートが正常に完了すると") {
		t.Error("HTMLBody does not describe successful replacement")
	}
	if !strings.Contains(htmlStr, `lang="ja"`) {
		t.Error("HTMLBody does not contain lang=ja")
	}

	textStr := renderComponent(t, ctx, sent.TextBody)
	wantText := "ポストのエクスポートが完了しました。\n\n" +
		"下記のページからダウンロードできます。\n\n" +
		exportURL + "\n\n" +
		"ダウンロードできるのは最新のエクスポート 1 件です。\n" +
		"新しいエクスポートが正常に完了すると、今回のエクスポートはダウンロードできなくなります。\n\n" +
		"-- \nMewst\nhttps://mewst.com\n"
	if textStr != wantText {
		t.Errorf("TextBody = %q, want %q", textStr, wantText)
	}
}

func TestExportCompletedSender_Send_English(t *testing.T) {
	t.Parallel()

	const exportURL = "https://mewst.com/settings/export"

	noopSender := NewNoopSender()
	sender := NewExportCompletedSender(noopSender)

	ctx := i18n.SetLocale(context.Background(), "en")

	err := sender.Send(ctx, "test@example.com", exportURL, "en", "01J000000000000000000EXPRT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sent := noopSender.SentEmails[0]

	if sent.Subject != "[Mewst] Your export is ready" {
		t.Errorf("Subject = %q, want %q", sent.Subject, "[Mewst] Your export is ready")
	}

	htmlStr := renderComponent(t, ctx, sent.HTMLBody)
	if !strings.Contains(htmlStr, exportURL) {
		t.Error("HTMLBody does not contain the export URL")
	}
	if !strings.Contains(htmlStr, ">Open your export page</a>") {
		t.Error("HTMLBody does not contain descriptive export link text")
	}
	if !strings.Contains(htmlStr, ">Mewst website</a>") {
		t.Error("HTMLBody does not contain descriptive website link text")
	}
	if !strings.Contains(htmlStr, "Once a newer export completes successfully") {
		t.Error("HTMLBody does not describe successful replacement")
	}
	if !strings.Contains(htmlStr, `lang="en"`) {
		t.Error("HTMLBody does not contain lang=en")
	}

	textStr := renderComponent(t, ctx, sent.TextBody)
	wantText := "Your posts have been exported.\n\n" +
		"You can download the archive from the page below.\n\n" +
		exportURL + "\n\n" +
		"Only your most recent export is available for download.\n" +
		"Once a newer export completes successfully, this export can no longer be downloaded.\n\n" +
		"-- \nMewst\nhttps://mewst.com\n"
	if textStr != wantText {
		t.Errorf("TextBody = %q, want %q", textStr, wantText)
	}
}

// TestExportCompletedSender_Send_UnsupportedLocale_FallsBackToJapanese pins that
// the subject and the body fall back together. The i18n matcher resolves a
// well-formed language tag it does not carry (such as "fr") to English, so
// without normalizing the locale first the mail would carry an English subject
// over a Japanese body.
//
// [Ja] TestExportCompletedSender_Send_UnsupportedLocale_FallsBackToJapanese は、
// 件名と本文が揃ってフォールバックすることを固定する。i18n のマッチャは、持って
// いない妥当な言語タグ ("fr" など) を英語へ解決するため、先にロケールを正規化
// しなければ、日本語の本文に英語の件名が乗ったメールになる。
func TestExportCompletedSender_Send_UnsupportedLocale_FallsBackToJapanese(t *testing.T) {
	t.Parallel()

	// The three inputs reach the fallback by different routes, so each case is
	// named rather than identified by its locale string ("" would otherwise be
	// reported as an index).
	//
	// [Ja] 3 つの入力はそれぞれ別の経路でフォールバックに至るため、ロケールの
	// 文字列ではなく名前で識別する ("" はそのままだと連番で報告されるため)。
	tests := []struct {
		name   string
		locale string
	}{
		{name: "言語タグとして解釈できない文字列", locale: "unknown"},
		{name: "妥当だがバンドルが持っていない言語タグ", locale: "fr"},
		{name: "呼び出し元が誤って渡しうるゼロ値", locale: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			noopSender := NewNoopSender()
			sender := NewExportCompletedSender(noopSender)

			err := sender.Send(context.Background(), "test@example.com", "https://mewst.com/settings/export", tt.locale, "01J000000000000000000EXPRT")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			sent := noopSender.SentEmails[0]

			if sent.Subject != "[Mewst] エクスポートの準備ができました" {
				t.Errorf("Subject = %q, want %q", sent.Subject, "[Mewst] エクスポートの準備ができました")
			}

			ctx := i18n.SetLocale(context.Background(), "ja")

			htmlStr := renderComponent(t, ctx, sent.HTMLBody)
			if !strings.Contains(htmlStr, `lang="ja"`) {
				t.Error("HTMLBody does not contain lang=ja")
			}

			textStr := renderComponent(t, ctx, sent.TextBody)
			if !strings.Contains(textStr, "ポストのエクスポートが完了しました。") {
				t.Error("TextBody is not the Japanese body")
			}
		})
	}
}

// TestExportCompletedSender_Send_OverridesALocalizerInTheContext pins that the
// locale argument decides the subject as well as the body. i18n.T prefers a
// localizer already in the context over the locale set beside it, so a caller
// context built by the HTTP i18n middleware would otherwise put a subject in
// the reader's browser language on a body in the recipient's stored language.
//
// [Ja] TestExportCompletedSender_Send_OverridesALocalizerInTheContext は、
// locale 引数が本文だけでなく件名も決めることを固定する。i18n.T は、その隣に
// 設定されたロケールよりも context に既にある Localizer を優先するため、HTTP の
// i18n ミドルウェアが組み立てた context を渡すと、宛先に保存された言語の本文に
// 閲覧者のブラウザ言語の件名が乗ってしまう。
func TestExportCompletedSender_Send_OverridesALocalizerInTheContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		contextLocale  string
		locale         string
		wantSubject    string
		wantLangAttr   string
		wantTextPrefix string
	}{
		{
			name:           "en の Localizer を持つ context に ja で送る",
			contextLocale:  i18n.LangEn,
			locale:         i18n.LangJa,
			wantSubject:    "[Mewst] エクスポートの準備ができました",
			wantLangAttr:   `lang="ja"`,
			wantTextPrefix: "ポストのエクスポートが完了しました。",
		},
		{
			name:           "ja の Localizer を持つ context に en で送る",
			contextLocale:  i18n.LangJa,
			locale:         i18n.LangEn,
			wantSubject:    "[Mewst] Your export is ready",
			wantLangAttr:   `lang="en"`,
			wantTextPrefix: "Your posts have been exported.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Build the context the way i18n.Middleware does, so that the
			// localizer and the locale are both present and disagree with the
			// locale the caller asks for.
			//
			// [Ja] i18n.Middleware と同じ形で context を組み立て、Localizer と
			// ロケールの両方が乗っていて、かつ呼び出し元が求めるロケールとは
			// 食い違っている状態にする。
			ctx := i18n.SetLocale(context.Background(), tt.contextLocale)
			ctx = i18n.SetLocalizer(ctx, i18n.NewLocalizer(tt.contextLocale))

			noopSender := NewNoopSender()
			sender := NewExportCompletedSender(noopSender)

			err := sender.Send(ctx, "test@example.com", "https://mewst.com/settings/export", tt.locale, "01J000000000000000000EXPRT")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			sent := noopSender.SentEmails[0]

			if sent.Subject != tt.wantSubject {
				t.Errorf("Subject = %q, want %q", sent.Subject, tt.wantSubject)
			}

			htmlStr := renderComponent(t, ctx, sent.HTMLBody)
			if !strings.Contains(htmlStr, tt.wantLangAttr) {
				t.Errorf("HTMLBody does not contain %s", tt.wantLangAttr)
			}

			textStr := renderComponent(t, ctx, sent.TextBody)
			if !strings.HasPrefix(textStr, tt.wantTextPrefix) {
				t.Errorf("TextBody = %q, want it to start with %q", textStr, tt.wantTextPrefix)
			}
		})
	}
}

// TestExportCompletedSender_Send_DoesNotEscapeTheTextBody pins that the
// text/plain part is delivered verbatim. templ escapes expressions for an HTML
// context, and applying that to the plain text part would put &amp; in front of
// every query parameter of the export URL.
//
// [Ja] TestExportCompletedSender_Send_DoesNotEscapeTheTextBody は、text/plain
// パートがそのまま配信されることを固定する。templ は式を HTML 文脈のために
// エスケープするが、それを plain text パートへ適用すると、エクスポート URL の
// すべてのクエリパラメータの前に &amp; が入ってしまう。
func TestExportCompletedSender_Send_DoesNotEscapeTheTextBody(t *testing.T) {
	t.Parallel()

	const exportURL = "https://mewst.com/settings/export?from=email&lang=ja"

	for _, locale := range []string{"ja", "en"} {
		t.Run(locale, func(t *testing.T) {
			t.Parallel()

			noopSender := NewNoopSender()
			sender := NewExportCompletedSender(noopSender)

			ctx := i18n.SetLocale(context.Background(), locale)

			err := sender.Send(ctx, "test@example.com", exportURL, locale, "01J000000000000000000EXPRT")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			textStr := renderComponent(t, ctx, noopSender.SentEmails[0].TextBody)

			if !strings.Contains(textStr, exportURL) {
				t.Errorf("TextBody does not contain the export URL verbatim: %q", textStr)
			}
			if strings.Contains(textStr, "&amp;") {
				t.Errorf("TextBody contains an HTML entity: %q", textStr)
			}
		})
	}
}

// TestExportCompletedSender_Send_DoesNotPromiseAnExpiry pins the requirement
// that the notification names no deadline. Retention is "the newest successful
// export", which no elapsed time decides, so a mail claiming one would be
// wrong the moment the reader believed it.
//
// [Ja] TestExportCompletedSender_Send_DoesNotPromiseAnExpiry は、通知が期限を
// 挙げないという要件を固定する。保持は「最新の成功したエクスポート」であり、
// 経過時間が決めるものではないため、期限を主張するメールは読み手がそれを信じた
// 時点で誤りになる。
func TestExportCompletedSender_Send_DoesNotPromiseAnExpiry(t *testing.T) {
	t.Parallel()

	expiryWords := map[string][]string{
		"ja": {"有効期限", "期限", "時間以内", "日以内"},
		"en": {"expire", "expires", "valid for", "within"},
	}

	for locale, words := range expiryWords {
		t.Run(locale, func(t *testing.T) {
			t.Parallel()

			noopSender := NewNoopSender()
			sender := NewExportCompletedSender(noopSender)

			ctx := i18n.SetLocale(context.Background(), locale)

			err := sender.Send(ctx, "test@example.com", "https://mewst.com/settings/export", locale, "01J000000000000000000EXPRT")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			sent := noopSender.SentEmails[0]

			for _, body := range []templ.Component{sent.HTMLBody, sent.TextBody} {
				rendered := renderComponent(t, ctx, body)
				for _, word := range words {
					if strings.Contains(rendered, word) {
						t.Errorf("body claims an expiry: contains %q", word)
					}
				}
			}
		})
	}
}

func TestExportCompletedSender_Send_MissingExportID(t *testing.T) {
	t.Parallel()

	noopSender := NewNoopSender()
	sender := NewExportCompletedSender(noopSender)

	err := sender.Send(context.Background(), "test@example.com", "https://mewst.com/settings/export", "ja", "")

	// Without the export ID every notification would share one idempotency
	// key, so the provider would deliver the first and drop the rest. Failing
	// closed keeps that silent loss out of the mail path.
	//
	// [Ja] エクスポート ID が無いと、すべての通知が 1 つの冪等キーを共有し、
	// プロバイダーは最初の 1 通を配信して残りを捨てる。fail-closed にすることで、
	// その静かな消失をメールの経路に入れない。
	if !errors.Is(err, ErrExportCompletedExportIDRequired) {
		t.Fatalf("error = %v, want %v", err, ErrExportCompletedExportIDRequired)
	}

	if len(noopSender.SentEmails) != 0 {
		t.Errorf("SentEmails count = %d, want 0", len(noopSender.SentEmails))
	}
}

func TestExportCompletedSender_Send_PropagatesSenderError(t *testing.T) {
	t.Parallel()

	sender := NewExportCompletedSender(&failingSender{})

	err := sender.Send(context.Background(), "test@example.com", "https://mewst.com/settings/export", "ja", "01J000000000000000000EXPRT")

	if !errors.Is(err, errSenderFailed) {
		t.Fatalf("error = %v, want it to wrap %v", err, errSenderFailed)
	}
}
