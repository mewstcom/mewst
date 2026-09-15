package email

import (
	"context"
	"errors"
	"fmt"

	"github.com/a-h/templ"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/templates/emails/export_completed"
)

// exportCompletedIdempotencyPrefixは完了通知の冪等キーに名前空間を与え、
// 後から冪等キーを持つ他の種類のメールのキーと区別されるようにする。
const exportCompletedIdempotencyPrefix = "export-completed/"

// ErrExportCompletedExportIDRequiredは、完了通知が、それが知らせる
// エクスポートを伴わずに求められたときに返る。
//
// 冪等キーを一意にしているのはエクスポートIDである。それ無しに送ると、
// すべてのエクスポートが同じキーを持つことになり、プロバイダーは最初の通知だけを
// 配信して残りを黙って捨てることになる。
var ErrExportCompletedExportIDRequired = errors.New("エクスポート完了メールにはエクスポートIDが必要です")

// ExportCompletedSenderはエクスポートが完了しダウンロードできるように
// なったことを知らせる通知を送る。
type ExportCompletedSender struct {
	sender Sender
}

// NewExportCompletedSenderはエクスポート完了メールのSenderを作成する。
func NewExportCompletedSender(sender Sender) *ExportCompletedSender {
	return &ExportCompletedSender{sender: sender}
}

// Sendは指定されたエクスポートの完了通知を描画して送る。
//
// exportURLはアーカイブそのものではなくエクスポート画面を指す。読み手を画面の
// 現在の状態を通させ、次のエクスポートが無効にするリンクを辿らせないため。
func (s *ExportCompletedSender) Send(ctx context.Context, to, exportURL, locale, exportID string) error {
	if exportID == "" {
		return ErrExportCompletedExportIDRequired
	}

	// 件名と本文を解決する前に正規化する。i18nのマッチャと下のswitchは
	// フォールバック先が異なる。"fr" のような妥当だがサポート外の言語タグは英語の
	// 件名に解決される一方、switchは日本語の本文を選ぶため、1通のメールが2つの
	// 言語で送られてしまう。
	if locale != i18n.LangEn {
		locale = i18n.LangJa
	}

	// ロケールと一緒にLocalizerも差し替える。i18n.Tはcontextに
	// Localizerがあればそれを読むため、Localizerを持つ呼び出し元では、本文が
	// locale引数に従う一方で、件名だけがそのcontextの言語で解決されてしまう。
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
