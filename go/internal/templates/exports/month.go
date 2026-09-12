package exports

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/a-h/templ"
)

// MonthData is what a month's file needs beyond the shared document shell.
//
// [Ja] MonthData は共通のドキュメントの外枠に加えて、月のファイルに必要なもの。
type MonthData struct {
	// Locale is the user's locale, decided the same way as for index.html.
	//
	// [Ja] Locale はユーザーのロケール。index.html と同じ扱い。
	Locale string

	// Title is the file's title, resolved by the caller because the shared
	// document shell takes it as text.
	//
	// [Ja] Title はそのファイルのタイトル。共通のドキュメントの外枠がテキスト
	// として受け取るため、呼び出し側が解決する。
	Title string

	// MonthStart names the month the file holds. It is a calendar label rather
	// than an instant, so it is read as written.
	//
	// [Ja] MonthStart はそのファイルが持つ月を指す。時点ではなく暦月のラベルの
	// ため、書かれたとおりに読む。
	MonthStart time.Time
}

// MonthPostData is one post written into a month's file.
//
// [Ja] MonthPostData は月のファイルへ書き出すポスト 1 件。
type MonthPostData struct {
	// ID is the post's identifier, written to data-post-id so that a parser can
	// tie an article back to the post it came from.
	//
	// [Ja] ID はポストの識別子。パーサーが article を元のポストへ辿れるよう
	// data-post-id へ書き出す。
	ID string

	// Content is the post body. It is written escaped, with its newlines kept
	// as they are, so that the original text comes back out of the element's
	// text content.
	//
	// [Ja] Content はポストの本文。エスケープし、改行をそのまま保って書き出す
	// ため、要素のテキストコンテンツから元の本文がそのまま得られる。
	Content string

	// PublishedAt is the instant the post was published, already converted to
	// the archive's zone by the caller.
	//
	// [Ja] PublishedAt はポストが投稿された時点。呼び出し側がアーカイブの
	// ゾーンへ変換済み。
	PublishedAt time.Time
}

const (
	// mainOpen and mainClose surround the posts of a month. A month's file is
	// written as a header, one fragment per post and a footer, so its main
	// element is opened and closed by different fragments and cannot be a
	// single templ element.
	//
	// [Ja] mainOpen と mainClose は月のポストを囲む。月のファイルはヘッダー →
	// ポストごとのフラグメント → フッターの順に書き出すため、その main 要素は
	// 別々のフラグメントが開閉することになり、1 つの templ 要素にできない。
	mainOpen  = "<main>\n"
	mainClose = "</main>"

	// commentFormat wraps the format comment. The spaces keep the text away
	// from the delimiters, so that a sentence which ends in a hyphen cannot run
	// into the closing one.
	//
	// [Ja] commentFormat は format のコメントを囲む。テキストと区切り記号の間の
	// 空白により、ハイフンで終わる文が終了の区切り記号と繋がらないようにする。
	commentFormat = "<!-- %s -->\n"
)

// MonthStart opens a month's file: the document shell index.html also uses,
// the comment describing the output contract, and the month's heading.
// MonthEnd closes it.
//
// The comment sits at the top of the body rather than above the doctype,
// because anything written before the doctype puts a browser into quirks mode
// and delays the encoding declaration.
//
// [Ja] MonthStart は月のファイルを開始する。index.html と共通のドキュメントの
// 外枠、出力契約を説明するコメント、月の見出しを書き出す。閉じるのは MonthEnd。
//
// コメントを doctype の上ではなく body の先頭に置くのは、doctype より前に何かを
// 書くとブラウザが quirks モードへ落ち、文字コード宣言も後ろへずれるため。
func MonthStart(data MonthData) templ.Component {
	return templ.Join(
		DocumentStart(DocumentData{Locale: data.Locale, Title: data.Title}),
		formatComment(),
		templ.Raw(mainOpen),
		monthHeading(data),
		templ.Raw("\n"),
	)
}

// MonthEnd closes the file MonthStart opened.
//
// [Ja] MonthEnd は MonthStart が開始したファイルを閉じる。
func MonthEnd() templ.Component {
	return templ.Join(templ.Raw(mainClose), DocumentEnd())
}

// MonthPost renders one post of a month's file. The fragment ends with a
// newline so that a file stays one post per line when it is opened in a text
// editor.
//
// [Ja] MonthPost は月のファイルのポストを 1 件描画する。テキストエディタで
// 開いたときにファイルが 1 行 1 ポストのままになるよう、フラグメントの末尾は
// 改行で終える。
func MonthPost(data MonthPostData) templ.Component {
	return templ.Join(monthPost(data), templ.Raw("\n"))
}

// formatComment writes the comment describing how a post is written. It is
// resolved while rendering rather than by the caller, because its text follows
// the reader's locale like the rest of the file.
//
// [Ja] formatComment はポストがどう書かれているかを説明するコメントを書き出す。
// 呼び出し側ではなく描画時に解決するのは、その文言もファイルの他の部分と同じく
// 読み手のロケールに従うため。
func formatComment() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, fmt.Sprintf(commentFormat, commentText(formatCommentText(ctx))))
		return err
	})
}

// commentText makes a text safe to place inside an HTML comment. The text
// comes from the translation files rather than from a user, but a comment is
// written raw, so the characters that could end it early or turn it into
// markup are taken out here instead of being left to whoever edits a
// translation.
//
// [Ja] commentText はテキストを HTML コメント内に置いても安全な形にする。
// テキストはユーザーではなく翻訳ファイル由来だが、コメントは raw で書き出す
// ため、コメントを途中で終わらせたりマークアップに変えたりしうる文字は、翻訳を
// 編集する人に委ねずここで取り除く。
func commentText(s string) string {
	s = strings.NewReplacer("<", "", ">", "").Replace(s)
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.TrimSpace(s)
}
