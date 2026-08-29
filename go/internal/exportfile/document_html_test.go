package exportfile_test

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// assertDocumentContract checks what every HTML file of an archive has to
// satisfy, whatever its content. index.html and the month files share one
// document shell, but a reader opens them as separate files, so the contract
// is asserted on each stored entry rather than on the shell alone.
//
// [Ja] assertDocumentContract は、内容に依らずアーカイブの各 HTML ファイルが
// 満たすべきことを検証する。index.html と月のファイルはドキュメントの外枠を
// 共有するが、読み手はそれぞれを別のファイルとして開くため、契約は外枠ではなく
// 格納された各エントリに対して検証する。
func assertDocumentContract(t *testing.T, body string, wantLang string) {
	t.Helper()

	// A byte order mark, a comment or even a blank line before the doctype
	// would put a browser into quirks mode, so the doctype is the first thing
	// in the file.
	//
	// [Ja] doctype より前の byte order mark・コメント・空行はブラウザを quirks
	// モードへ落とすため、doctype をファイルの先頭に置く。
	if !strings.HasPrefix(body, "<!doctype html>") {
		t.Errorf("ファイルが doctype で始まっていない: %s", body)
	}
	if !strings.HasSuffix(body, "</html>\n") {
		t.Errorf("ファイルが閉じられていない: %s", body)
	}

	// The files are opened from the local file system, where there is no
	// Content-Type header, so the encoding declaration has to be inside the
	// first 1024 bytes a browser sniffs.
	//
	// [Ja] ファイルはローカルファイルシステムから開かれ Content-Type ヘッダーが
	// 無いため、文字コード宣言はブラウザが読む先頭 1024 バイトに収める必要がある。
	const charsetMeta = `<meta charset="utf-8">`
	charsetAt := strings.Index(body, charsetMeta)
	if charsetAt < 0 {
		t.Fatalf("ファイルに %q が含まれていない: %s", charsetMeta, body)
	}
	if end := charsetAt + len(charsetMeta); end > 1024 {
		t.Errorf("文字コード宣言の終端 = %d バイト目, want 1024 バイト以内", end)
	}

	document := parseHTML(t, body)
	head := findElement(t, document, "head")
	metas := findElements(head, "meta")
	if len(metas) == 0 || metas[0] != head.FirstChild {
		t.Errorf("charset の meta が head の最初の子ではない: %s", body)
	}
	if got := attribute(metas[0], "charset"); got != "utf-8" {
		t.Errorf("head の最初の子の charset = %q, want %q", got, "utf-8")
	}

	if got := attribute(findElement(t, document, "html"), "lang"); got != wantLang {
		t.Errorf("html[lang] = %q, want %q", got, wantLang)
	}

	// Zoom must stay available: someone reading their own archive on a phone
	// relies on it (WCAG 1.4.4).
	//
	// [Ja] ズームは使えるままにする。スマートフォンで自分のアーカイブを読む人が
	// ズームに頼るため (WCAG 1.4.4)。
	var viewport, colorScheme, formatVersion string
	for _, meta := range findElements(document, "meta") {
		switch attribute(meta, "name") {
		case "viewport":
			viewport = attribute(meta, "content")
		case "color-scheme":
			colorScheme = attribute(meta, "content")
		case "mewst-export-format":
			formatVersion = attribute(meta, "content")
		}
	}
	if viewport != "width=device-width, initial-scale=1" {
		t.Errorf("viewport = %q, want %q", viewport, "width=device-width, initial-scale=1")
	}
	// The declared colour schemes have to match what the stylesheet actually
	// supports, so a reader in dark mode gets dark browser chrome as the file
	// opens instead of a white flash.
	//
	// [Ja] 宣言する配色はスタイルシートが実際に対応する配色と一致させる。
	// ダークモードの読み手が、ファイルを開いた瞬間に白く光らせず、ブラウザの
	// 描画もダークで受け取れるようにするため。
	if colorScheme != "light dark" {
		t.Errorf("color-scheme = %q, want %q", colorScheme, "light dark")
	}
	if !strings.Contains(body, "prefers-color-scheme: dark") {
		t.Errorf("スタイルにダークモードの配色が無い: %s", body)
	}
	// A parser is handed one file at a time, so every file names the contract
	// it follows.
	//
	// [Ja] パーサーが渡されるのは 1 ファイル単位のため、各ファイルが自身の従う
	// 契約を示す。
	if formatVersion != "1" {
		t.Errorf("mewst-export-format = %q, want %q", formatVersion, "1")
	}

	if len(findElements(document, "title")) != 1 {
		t.Errorf("title が 1 つではない: %s", body)
	}
	if len(findElements(document, "style")) != 1 {
		t.Errorf("インラインのスタイルシートが 1 つではない: %s", body)
	}
	// The archive is opened offline from a file manager, and iOS Quick Look
	// does not run scripts, so no file may depend on one.
	//
	// [Ja] アーカイブはファイルアプリからオフラインで開かれ、iOS のクイックルック
	// はスクリプトを実行しないため、どのファイルもスクリプトに依存してはならない。
	if scripts := findElements(document, "script"); len(scripts) != 0 {
		t.Errorf("ファイルにスクリプトが含まれている: %s", body)
	}
}

// firstMeaningfulChild returns a node's first child that is not the whitespace
// the markup is laid out with, so that a test can say what comes first without
// depending on where the fragments break lines.
//
// [Ja] firstMeaningfulChild は、マークアップの体裁のための空白ではない最初の子を
// 返す。フラグメントがどこで改行するかに依らず、テストが「最初に来るもの」を
// 言えるようにするため。
func firstMeaningfulChild(node *html.Node) *html.Node {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.TextNode && strings.TrimSpace(child.Data) == "" {
			continue
		}
		return child
	}
	return nil
}

// findComments returns every HTML comment of a document, in document order.
//
// [Ja] findComments はドキュメント内の HTML コメントをすべて、ドキュメント順で
// 返す。
func findComments(node *html.Node) []*html.Node {
	var found []*html.Node
	if node.Type == html.CommentNode {
		found = append(found, node)
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		found = append(found, findComments(child)...)
	}
	return found
}
