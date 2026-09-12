package email

import (
	"context"
	"errors"
	"fmt"

	"github.com/a-h/templ"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/templates/emails/export_completed"
)

// exportCompletedIdempotencyPrefix namespaces the idempotency key of the
// completion notification, so that a key stays distinct from the keys of other
// mail kinds that may adopt one later.
//
// [Ja] exportCompletedIdempotencyPrefix は完了通知の冪等キーに名前空間を与え、
// 後から冪等キーを持つ他の種類のメールのキーと区別されるようにする。
const exportCompletedIdempotencyPrefix = "export-completed/"

// ErrExportCompletedExportIDRequired is returned when the completion
// notification is asked for without the export it announces.
//
// The export ID is what makes the idempotency key unique. Sending without one
// would give every export the same key, and the provider would deliver only the
// first notification and silently drop the rest.
//
// [Ja] ErrExportCompletedExportIDRequired は、完了通知が、それが知らせる
// エクスポートを伴わずに求められたときに返る。
//
// 冪等キーを一意にしているのはエクスポート ID である。それ無しに送ると、
// すべてのエクスポートが同じキーを持つことになり、プロバイダーは最初の通知だけを
// 配信して残りを黙って捨てることになる。
var ErrExportCompletedExportIDRequired = errors.New("エクスポート完了メールにはエクスポート ID が必要です")

// ExportCompletedSender sends the notification that an export finished and is
// ready to download.
//
// [Ja] ExportCompletedSender はエクスポートが完了しダウンロードできるように
// なったことを知らせる通知を送る。
type ExportCompletedSender struct {
	sender Sender
}

// NewExportCompletedSender creates an export completion email sender.
//
// [Ja] NewExportCompletedSender はエクスポート完了メールの Sender を作成する。
func NewExportCompletedSender(sender Sender) *ExportCompletedSender {
	return &ExportCompletedSender{sender: sender}
}

// Send renders and sends the completion notification for the given export.
//
// exportURL points at the export screen rather than the archive itself, so that
// the reader passes through the screen's current state instead of following a
// link that the next export invalidates.
//
// [Ja] Send は指定されたエクスポートの完了通知を描画して送る。
//
// exportURL はアーカイブそのものではなくエクスポート画面を指す。読み手を画面の
// 現在の状態を通させ、次のエクスポートが無効にするリンクを辿らせないため。
func (s *ExportCompletedSender) Send(ctx context.Context, to, exportURL, locale, exportID string) error {
	if exportID == "" {
		return ErrExportCompletedExportIDRequired
	}

	// Normalize before the subject and the body are resolved. The i18n matcher
	// and the switch below fall back differently: a well-formed but unsupported
	// language tag such as "fr" resolves to the English subject while the switch
	// picks the Japanese body, which would send one mail in two languages.
	//
	// [Ja] 件名と本文を解決する前に正規化する。i18n のマッチャと下の switch は
	// フォールバック先が異なる。"fr" のような妥当だがサポート外の言語タグは英語の
	// 件名に解決される一方、switch は日本語の本文を選ぶため、1 通のメールが 2 つの
	// 言語で送られてしまう。
	if locale != i18n.LangEn {
		locale = i18n.LangJa
	}

	// Replace the localizer along with the locale. i18n.T reads the localizer
	// already in the context when there is one, so a caller that carries one
	// would resolve the subject in its own language while the body follows the
	// locale argument.
	//
	// [Ja] ロケールと一緒に Localizer も差し替える。i18n.T は context に
	// Localizer があればそれを読むため、Localizer を持つ呼び出し元では、本文が
	// locale 引数に従う一方で、件名だけがその context の言語で解決されてしまう。
	ctx = i18n.SetLocale(ctx, locale)
	ctx = i18n.SetLocalizer(ctx, i18n.NewLocalizer(locale))

	subject := i18n.T(ctx, "export_completed_email_subject")

	var htmlBody, textBody templ.Component
	switch locale {
	case i18n.LangEn:
		htmlBody = export_completed.EnHTML(exportURL)
		textBody = export_completed.EnText(exportURL)
	default:
		htmlBody = export_completed.JaHTML(exportURL)
		textBody = export_completed.JaText(exportURL)
	}

	if err := s.sender.Send(ctx, SendInput{
		To:             to,
		Subject:        subject,
		HTMLBody:       htmlBody,
		TextBody:       textBody,
		IdempotencyKey: exportCompletedIdempotencyPrefix + exportID,
	}); err != nil {
		return fmt.Errorf("エクスポート完了メールの送信に失敗: %w", err)
	}

	return nil
}
