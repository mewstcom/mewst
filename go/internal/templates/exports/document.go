// Package exports renders the HTML files an export archive is made of:
// index.html and one file per month. The files are opened from a file manager
// after the zip is unpacked, so they are self-contained: no scripts, no
// external stylesheets and no other network access.
//
// [Ja] exports パッケージはエクスポートアーカイブを構成する HTML ファイル
// (index.html と月ごとのファイル) を描画する。これらのファイルは zip を解凍した
// 後にファイルアプリから開かれるため、スクリプト・外部スタイルシート・その他の
// ネットワークアクセスを持たない自己完結した構成にする。
package exports

import (
	"fmt"

	"github.com/a-h/templ"

	"github.com/mewstcom/mewst/go/internal/i18n"
)

// FormatVersion is the version of the output contract the archive's HTML
// follows. It is recorded in every file so that a parser written against this
// archive can tell which contract a file it is given follows.
//
// [Ja] FormatVersion はアーカイブの HTML が従う出力契約のバージョン。各ファイルへ
// 記録することで、このアーカイブ向けに書かれたパーサーが、渡されたファイルが
// どの契約に従うかを判別できるようにする。
const FormatVersion = "1"

// DocumentData is what every HTML file of an archive needs: the locale its own
// text is written in, and the title of that file.
//
// [Ja] DocumentData はアーカイブの各 HTML ファイルに必要なもの、すなわちその
// ファイル自身の文言を書くロケールと、そのファイルのタイトル。
type DocumentData struct {
	// Locale is the user's locale. It decides both the translations and the
	// language tag written to html[lang].
	//
	// [Ja] Locale はユーザーのロケール。翻訳と html[lang] に書く言語タグの
	// どちらもこれで決まる。
	Locale string

	// Title is the file's title, shown in the browser tab and by file previews.
	//
	// [Ja] Title はそのファイルのタイトル。ブラウザのタブやファイルの
	// プレビューに表示される。
	Title string
}

const (
	// documentOpenFormat opens the document. Nothing is written before the
	// doctype, so no byte order mark or comment can push a browser into quirks
	// mode or delay the encoding declaration.
	//
	// [Ja] documentOpenFormat はドキュメントを開始する。doctype より前には何も
	// 書き出さない。byte order mark やコメントがブラウザを quirks モードへ
	// 落としたり、文字コード宣言を後ろへずらしたりしないようにするため。
	documentOpenFormat = "<!doctype html>\n<html lang=\"%s\">\n"

	// bodyOpen and documentClose surround a file's content.
	//
	// [Ja] bodyOpen と documentClose はファイルの内容を囲む。
	bodyOpen      = "\n<body>\n"
	documentClose = "\n</body>\n</html>\n"
)

// DocumentStart opens an HTML file of the archive: the doctype, the html
// element and the head, up to the start of the body. DocumentEnd closes it.
//
// The document is split in halves rather than written as one component
// because a month's file is streamed (its header, one fragment per post, then
// its footer) while templ markup has to be balanced. Keeping both halves here
// lets index.html and the month files share the same document shell.
//
// [Ja] DocumentStart はアーカイブの HTML ファイルを開始する。doctype・html 要素・
// head を書き出し、body の開始までを担う。閉じるのは DocumentEnd。
//
// ドキュメントを 1 つのコンポーネントにせず前半と後半に分けているのは、月の
// ファイルがストリーミングで書き出される (ヘッダー → 投稿ごとのフラグメント →
// フッター) 一方で、templ のマークアップは閉じている必要があるため。両方を
// ここに置くことで、index.html と月のファイルが同じドキュメントの外枠を共有できる。
func DocumentStart(data DocumentData) templ.Component {
	return templ.Join(
		templ.Raw(fmt.Sprintf(documentOpenFormat, langForLocale(data.Locale))),
		documentHead(data),
		templ.Raw(bodyOpen),
	)
}

// DocumentEnd closes the document DocumentStart opened.
//
// [Ja] DocumentEnd は DocumentStart が開始したドキュメントを閉じる。
func DocumentEnd() templ.Component {
	return templ.Raw(documentClose)
}

// langForLocale returns the BCP 47 language tag for a user locale. The result
// is one of a closed set of constants, so DocumentStart can write it into the
// lang attribute directly. An unknown locale falls back to the default
// language, which is also the language its translations resolve to.
//
// [Ja] langForLocale はユーザーのロケールに対応する BCP 47 言語タグを返す。
// 戻り値は決まった定数のいずれかのため、DocumentStart はそれをそのまま lang
// 属性へ書き出せる。未知のロケールは既定の言語へフォールバックする。翻訳も
// 同じ言語に解決されるため。
func langForLocale(locale string) string {
	if locale == i18n.LangEn {
		return i18n.LangEn
	}
	return i18n.LangJa
}
