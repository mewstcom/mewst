package exports

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/a-h/templ"
)

// MonthDataは共通のドキュメントの外枠に加えて、月のファイルに必要なもの。
type MonthData struct {
	// Localeはユーザーのロケール。index.htmlと同じ扱い。
	Locale string

	// Titleはそのファイルのタイトル。共通のドキュメントの外枠がテキスト
	// として受け取るため、呼び出し側が解決する。
	Title string

	// MonthStartはそのファイルが持つ月を指す。時点ではなく暦月のラベルの
	// ため、書かれたとおりに読む。
	MonthStart time.Time
}

// MonthPostDataは月のファイルへ書き出すポスト1件。
type MonthPostData struct {
	// IDはポストの識別子。パーサーがarticleを元のポストへ辿れるよう
	// data-post-idへ書き出す。
	ID string

	// Contentはポストの本文。エスケープし、改行をそのまま保って書き出す
	// ため、要素のテキストコンテンツから元の本文がそのまま得られる。
	Content string

	// PublishedAtはポストが投稿された時点。呼び出し側がアーカイブの
	// ゾーンへ変換済み。
	PublishedAt time.Time
}

const (
	// mainOpenとmainCloseは月のポストを囲む。月のファイルはヘッダー →
	// ポストごとのフラグメント → フッターの順に書き出すため、そのmain要素は
	// 別々のフラグメントが開閉することになり、1つのtempl要素にできない。
	mainOpen  = "<main>\n"
	mainClose = "</main>"

	// commentFormatはformatのコメントを囲む。テキストと区切り記号の間の
	// 空白により、ハイフンで終わる文が終了の区切り記号と繋がらないようにする。
	commentFormat = "<!-- %s -->\n"
)

// MonthStartは月のファイルを開始する。index.htmlと共通のドキュメントの
// 外枠、出力契約を説明するコメント、月の見出しを書き出す。閉じるのはMonthEnd。
//
// コメントをdoctypeの上ではなくbodyの先頭に置くのは、doctypeより前に何かを
// 書くとブラウザがquirksモードへ落ち、文字コード宣言も後ろへずれるため。
func MonthStart(data MonthData) templ.Component {
	return templ.Join(
		DocumentStart(DocumentData{Locale: data.Locale, Title: data.Title}),
		formatComment(),
		templ.Raw(mainOpen),
		monthHeading(data),
		templ.Raw("\n"),
	)
}

// MonthEndはMonthStartが開始したファイルを閉じる。
func MonthEnd() templ.Component {
	return templ.Join(templ.Raw(mainClose), DocumentEnd())
}

// MonthPostは月のファイルのポストを1件描画する。テキストエディタで
// 開いたときにファイルが1行1ポストのままになるよう、フラグメントの末尾は
// 改行で終える。
func MonthPost(data MonthPostData) templ.Component {
	return templ.Join(monthPost(data), templ.Raw("\n"))
}

// formatCommentはポストがどう書かれているかを説明するコメントを書き出す。
// 呼び出し側ではなく描画時に解決するのは、その文言もファイルの他の部分と同じく
// 読み手のロケールに従うため。
func formatComment() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, fmt.Sprintf(commentFormat, commentText(formatCommentText(ctx))))
		return err
	})
}

// commentTextはテキストをHTMLコメント内に置いても安全な形にする。
// テキストはユーザーではなく翻訳ファイル由来だが、コメントはrawで書き出す
// ため、コメントを途中で終わらせたりマークアップに変えたりしうる文字は、翻訳を
// 編集する人に委ねずここで取り除く。
func commentText(s string) string {
	s = strings.NewReplacer("<", "", ">", "").Replace(s)
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.TrimSpace(s)
}
