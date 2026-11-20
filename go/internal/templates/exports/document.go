// Package exportsはエクスポートアーカイブを構成するHTMLファイル
// (index.htmlと月ごとのファイル) を描画する。これらのファイルはzipを解凍した
// 後にファイルアプリから開かれるため、スクリプト・外部スタイルシート・その他の
// ネットワークアクセスを持たない自己完結した構成にする。
package exports

import (
	"fmt"

	"github.com/a-h/templ"

	"github.com/mewstcom/mewst/go/internal/i18n"
)

// FormatVersionはアーカイブのHTMLが従う出力契約のバージョン。各ファイルへ
// 記録することで、このアーカイブ向けに書かれたパーサーが、渡されたファイルが
// どの契約に従うかを判別できるようにする。
const FormatVersion = "1"

// DocumentDataはアーカイブの各HTMLファイルに必要なもの、すなわちその
// ファイル自身の文言を書くロケールと、そのファイルのタイトル。
type DocumentData struct {
	// Localeはユーザーのロケール。翻訳とhtml[lang] に書く言語タグの
	// どちらもこれで決まる。
	Locale string

	// Titleはそのファイルのタイトル。ブラウザのタブやファイルの
	// プレビューに表示される。
	Title string
}

const (
	// documentOpenFormatはドキュメントを開始する。doctypeより前には何も
	// 書き出さない。byte order markやコメントがブラウザをquirksモードへ
	// 落としたり、文字コード宣言を後ろへずらしたりしないようにするため。
	documentOpenFormat = "<!doctype html>\n<html lang=\"%s\">\n"

	// bodyOpenとdocumentCloseはファイルの内容を囲む。
	bodyOpen      = "\n<body>\n"
	documentClose = "\n</body>\n</html>\n"
)

// DocumentStartはアーカイブのHTMLファイルを開始する。doctype・html要素・
// headを書き出し、bodyの開始までを担う。閉じるのはDocumentEnd。
//
// ドキュメントを1つのコンポーネントにせず前半と後半に分けているのは、月の
// ファイルがストリーミングで書き出される (ヘッダー → 投稿ごとのフラグメント →
// フッター) 一方で、templのマークアップは閉じている必要があるため。両方を
// ここに置くことで、index.htmlと月のファイルが同じドキュメントの外枠を共有できる。
func DocumentStart(data DocumentData) templ.Component {
	return templ.Join(
		templ.Raw(fmt.Sprintf(documentOpenFormat, langForLocale(data.Locale))),
		documentHead(data),
		templ.Raw(bodyOpen),
	)
}

// DocumentEndはDocumentStartが開始したドキュメントを閉じる。
func DocumentEnd() templ.Component {
	return templ.Raw(documentClose)
}

// langForLocaleはユーザーのロケールに対応するBCP 47言語タグを返す。
// 戻り値は決まった定数のいずれかのため、DocumentStartはそれをそのままlang
// 属性へ書き出せる。未知のロケールは既定の言語へフォールバックする。翻訳も
// 同じ言語に解決されるため。
func langForLocale(locale string) string {
	if locale == i18n.LangEn {
		return i18n.LangEn
	}
	return i18n.LangJa
}
