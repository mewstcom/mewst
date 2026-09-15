package email

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/mewstcom/mewst/go/internal/i18n"
)

// errSenderFailedは下位のSenderの配信失敗を代弁する。
var errSenderFailed = errors.New("送信に失敗しました")

// failingSenderはすべての送信に失敗する。メールのプロバイダーへ到達せずに
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
		t.Fatalf("予期しないエラー: %v", err)
	}

	if len(noopSender.SentEmails) != 1 {
		t.Fatalf("SentEmailsの件数 = %d、期待値 = 1", len(noopSender.SentEmails))
	}

	sent := noopSender.SentEmails[0]

	if sent.To != "test@example.com" {
		t.Errorf("To = %q、期待値 = %q", sent.To, "test@example.com")
	}

	if sent.Subject != "[Mewst] エクスポートの準備ができました" {
		t.Errorf("Subject = %q、期待値 = %q", sent.Subject, "[Mewst] エクスポートの準備ができました")
	}

	// キーはエクスポートから導出する。応答を失った後に再試行されたジョブが
	// 同じ通知を1度だけ配信するため。
	if sent.IdempotencyKey != "export-completed/01J000000000000000000EXPRT" {
		t.Errorf("IdempotencyKey = %q、期待値 = %q", sent.IdempotencyKey, "export-completed/01J000000000000000000EXPRT")
	}

	htmlStr := renderComponent(t, ctx, sent.HTMLBody)
	if !strings.Contains(htmlStr, exportURL) {
		t.Error("HTMLBodyにエクスポートのURLが含まれていない")
	}
	if !strings.Contains(htmlStr, ">エクスポート画面を開く</a>") {
		t.Error("HTMLBodyにエクスポートへのリンクを説明するテキストが含まれていない")
	}
	if !strings.Contains(htmlStr, ">Mewst公式サイト</a>") {
		t.Error("HTMLBodyにウェブサイトへのリンクを説明するテキストが含まれていない")
	}
	if !strings.Contains(htmlStr, "新しいエクスポートが正常に完了すると") {
		t.Error("HTMLBodyに置き換えが成功した旨の説明が含まれていない")
	}
	if !strings.Contains(htmlStr, `lang="ja"`) {
		t.Error("HTMLBodyにlang=jaが含まれていない")
	}

	textStr := renderComponent(t, ctx, sent.TextBody)
	wantText := "ポストのエクスポートが完了しました。\n\n" +
		"下記のページからダウンロードできます。\n\n" +
		exportURL + "\n\n" +
		"ダウンロードできるのは最新のエクスポート1件です。\n" +
		"新しいエクスポートが正常に完了すると、今回のエクスポートはダウンロードできなくなります。\n\n" +
		"-- \nMewst\nhttps://mewst.com\n"
	if textStr != wantText {
		t.Errorf("TextBody = %q、期待値 = %q", textStr, wantText)
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
		t.Fatalf("予期しないエラー: %v", err)
	}

	sent := noopSender.SentEmails[0]

	if sent.Subject != "[Mewst] Your export is ready" {
		t.Errorf("Subject = %q、期待値 = %q", sent.Subject, "[Mewst] Your export is ready")
	}

	htmlStr := renderComponent(t, ctx, sent.HTMLBody)
	if !strings.Contains(htmlStr, exportURL) {
		t.Error("HTMLBodyにエクスポートのURLが含まれていない")
	}
	if !strings.Contains(htmlStr, ">Open your export page</a>") {
		t.Error("HTMLBodyにエクスポートへのリンクを説明するテキストが含まれていない")
	}
	if !strings.Contains(htmlStr, ">Mewst website</a>") {
		t.Error("HTMLBodyにウェブサイトへのリンクを説明するテキストが含まれていない")
	}
	if !strings.Contains(htmlStr, "Once a newer export completes successfully") {
		t.Error("HTMLBodyに置き換えが成功した旨の説明が含まれていない")
	}
	if !strings.Contains(htmlStr, `lang="en"`) {
		t.Error("HTMLBodyにlang=enが含まれていない")
	}

	textStr := renderComponent(t, ctx, sent.TextBody)
	wantText := "Your posts have been exported.\n\n" +
		"You can download the archive from the page below.\n\n" +
		exportURL + "\n\n" +
		"Only your most recent export is available for download.\n" +
		"Once a newer export completes successfully, this export can no longer be downloaded.\n\n" +
		"-- \nMewst\nhttps://mewst.com\n"
	if textStr != wantText {
		t.Errorf("TextBody = %q、期待値 = %q", textStr, wantText)
	}
}

// TestExportCompletedSender_Send_UnsupportedLocale_FallsBackToJapaneseは、
// 件名と本文が揃ってフォールバックすることを固定する。i18nのマッチャは、持って
// いない妥当な言語タグ ("fr" など) を英語へ解決するため、先にロケールを正規化
// しなければ、日本語の本文に英語の件名が乗ったメールになる。
func TestExportCompletedSender_Send_UnsupportedLocale_FallsBackToJapanese(t *testing.T) {
	t.Parallel()

	// 3つの入力はそれぞれ別の経路でフォールバックに至るため、ロケールの
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
				t.Fatalf("予期しないエラー: %v", err)
			}

			sent := noopSender.SentEmails[0]

			if sent.Subject != "[Mewst] エクスポートの準備ができました" {
				t.Errorf("Subject = %q、期待値 = %q", sent.Subject, "[Mewst] エクスポートの準備ができました")
			}

			ctx := i18n.SetLocale(context.Background(), "ja")

			htmlStr := renderComponent(t, ctx, sent.HTMLBody)
			if !strings.Contains(htmlStr, `lang="ja"`) {
				t.Error("HTMLBodyにlang=jaが含まれていない")
			}

			textStr := renderComponent(t, ctx, sent.TextBody)
			if !strings.Contains(textStr, "ポストのエクスポートが完了しました。") {
				t.Error("TextBodyが日本語の本文ではない")
			}
		})
	}
}

// TestExportCompletedSender_Send_OverridesALocalizerInTheContextは、
// locale引数が本文だけでなく件名も決めることを固定する。i18n.Tは、その隣に
// 設定されたロケールよりもcontextに既にあるLocalizerを優先するため、HTTPの
// i18nミドルウェアが組み立てたcontextを渡すと、宛先に保存された言語の本文に
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
			name:           "enのLocalizerを持つcontextにjaで送る",
			contextLocale:  i18n.LangEn,
			locale:         i18n.LangJa,
			wantSubject:    "[Mewst] エクスポートの準備ができました",
			wantLangAttr:   `lang="ja"`,
			wantTextPrefix: "ポストのエクスポートが完了しました。",
		},
		{
			name:           "jaのLocalizerを持つcontextにenで送る",
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

			// i18n.Middlewareと同じ形でcontextを組み立て、Localizerと
			// ロケールの両方が乗っていて、かつ呼び出し元が求めるロケールとは
			// 食い違っている状態にする。
			ctx := i18n.SetLocale(context.Background(), tt.contextLocale)
			ctx = i18n.SetLocalizer(ctx, i18n.NewLocalizer(tt.contextLocale))

			noopSender := NewNoopSender()
			sender := NewExportCompletedSender(noopSender)

			err := sender.Send(ctx, "test@example.com", "https://mewst.com/settings/export", tt.locale, "01J000000000000000000EXPRT")
			if err != nil {
				t.Fatalf("予期しないエラー: %v", err)
			}

			sent := noopSender.SentEmails[0]

			if sent.Subject != tt.wantSubject {
				t.Errorf("Subject = %q、期待値 = %q", sent.Subject, tt.wantSubject)
			}

			htmlStr := renderComponent(t, ctx, sent.HTMLBody)
			if !strings.Contains(htmlStr, tt.wantLangAttr) {
				t.Errorf("HTMLBodyに%sが含まれていない", tt.wantLangAttr)
			}

			textStr := renderComponent(t, ctx, sent.TextBody)
			if !strings.HasPrefix(textStr, tt.wantTextPrefix) {
				t.Errorf("TextBody = %q、%qで始まることを期待", textStr, tt.wantTextPrefix)
			}
		})
	}
}

// TestExportCompletedSender_Send_DoesNotEscapeTheTextBodyは、text/plain
// パートがそのまま配信されることを固定する。templは式をHTML文脈のために
// エスケープするが、それをplain textパートへ適用すると、エクスポートURLの
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
				t.Fatalf("予期しないエラー: %v", err)
			}

			textStr := renderComponent(t, ctx, noopSender.SentEmails[0].TextBody)

			if !strings.Contains(textStr, exportURL) {
				t.Errorf("TextBodyにエクスポートのURLがそのまま含まれていない: %q", textStr)
			}
			if strings.Contains(textStr, "&amp;") {
				t.Errorf("TextBodyにHTMLエンティティが含まれている: %q", textStr)
			}
		})
	}
}

// TestExportCompletedSender_Send_DoesNotPromiseAnExpiryは、通知が期限を
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
				t.Fatalf("予期しないエラー: %v", err)
			}

			sent := noopSender.SentEmails[0]

			for _, body := range []templ.Component{sent.HTMLBody, sent.TextBody} {
				rendered := renderComponent(t, ctx, body)
				for _, word := range words {
					if strings.Contains(rendered, word) {
						t.Errorf("本文が有効期限を示している: %qを含む", word)
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

	// エクスポートIDが無いと、すべての通知が1つの冪等キーを共有し、
	// プロバイダーは最初の1通を配信して残りを捨てる。fail-closedにすることで、
	// その静かな消失をメールの経路に入れない。
	if !errors.Is(err, ErrExportCompletedExportIDRequired) {
		t.Fatalf("error = %v、期待値 = %v", err, ErrExportCompletedExportIDRequired)
	}

	if len(noopSender.SentEmails) != 0 {
		t.Errorf("SentEmailsの件数 = %d、期待値 = 0", len(noopSender.SentEmails))
	}
}

func TestExportCompletedSender_Send_PropagatesSenderError(t *testing.T) {
	t.Parallel()

	sender := NewExportCompletedSender(&failingSender{})

	err := sender.Send(context.Background(), "test@example.com", "https://mewst.com/settings/export", "ja", "01J000000000000000000EXPRT")

	if !errors.Is(err, errSenderFailed) {
		t.Fatalf("エラー = %v、%vをラップしたエラーを期待", err, errSenderFailed)
	}
}
