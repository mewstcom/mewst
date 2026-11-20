package exportfile_test

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// assertDocumentContractは、内容に依らずアーカイブの各HTMLファイルが
// 満たすべきことを検証する。index.htmlと月のファイルはドキュメントの外枠を
// 共有するが、読み手はそれぞれを別のファイルとして開くため、契約は外枠ではなく
// 格納された各エントリに対して検証する。
func assertDocumentContract(t *testing.T, body string, wantLang string) {
	t.Helper()

	// doctypeより前のbyte order mark・コメント・空行はブラウザをquirks
	// モードへ落とすため、doctypeをファイルの先頭に置く。
	if !strings.HasPrefix(body, "<!doctype html>") {
		t.Errorf("ファイルがdoctypeで始まっていない: %s", body)
	}
	if !strings.HasSuffix(body, "</html>\n") {
		t.Errorf("ファイルが閉じられていない: %s", body)
	}

	// ファイルはローカルファイルシステムから開かれContent-Typeヘッダーが
	// 無いため、文字コード宣言はブラウザが読む先頭1024バイトに収める必要がある。
	const charsetMeta = `<meta charset="utf-8">`
	charsetAt := strings.Index(body, charsetMeta)
	if charsetAt < 0 {
		t.Fatalf("ファイルに%qが含まれていない: %s", charsetMeta, body)
	}
	if end := charsetAt + len(charsetMeta); end > 1024 {
		t.Errorf("文字コード宣言の終端 = %dバイト目、期待値 = 1024バイト以内", end)
	}

	document := parseHTML(t, body)
	head := findElement(t, document, "head")
	metas := findElements(head, "meta")
	if len(metas) == 0 || metas[0] != head.FirstChild {
		t.Errorf("charsetのmetaがheadの最初の子ではない: %s", body)
	}
	if got := attribute(metas[0], "charset"); got != "utf-8" {
		t.Errorf("headの最初の子のcharset = %q、期待値 = %q", got, "utf-8")
	}

	if got := attribute(findElement(t, document, "html"), "lang"); got != wantLang {
		t.Errorf("html[lang] = %q、期待値 = %q", got, wantLang)
	}

	// ズームは使えるままにする。スマートフォンで自分のアーカイブを読む人が
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
		t.Errorf("viewport = %q、期待値 = %q", viewport, "width=device-width, initial-scale=1")
	}
	// 宣言する配色はスタイルシートが実際に対応する配色と一致させる。
	// ダークモードの読み手が、ファイルを開いた瞬間に白く光らせず、ブラウザの
	// 描画もダークで受け取れるようにするため。
	if colorScheme != "light dark" {
		t.Errorf("color-scheme = %q、期待値 = %q", colorScheme, "light dark")
	}
	if !strings.Contains(body, "prefers-color-scheme: dark") {
		t.Errorf("スタイルにダークモードの配色が無い: %s", body)
	}
	// パーサーが渡されるのは1ファイル単位のため、各ファイルが自身の従う
	// 契約を示す。
	if formatVersion != "1" {
		t.Errorf("mewst-export-format = %q、期待値 = %q", formatVersion, "1")
	}

	if len(findElements(document, "title")) != 1 {
		t.Errorf("titleが1つではない: %s", body)
	}
	if len(findElements(document, "style")) != 1 {
		t.Errorf("インラインのスタイルシートが1つではない: %s", body)
	}
	// アーカイブはファイルアプリからオフラインで開かれ、iOSのクイックルック
	// はスクリプトを実行しないため、どのファイルもスクリプトに依存してはならない。
	if scripts := findElements(document, "script"); len(scripts) != 0 {
		t.Errorf("ファイルにスクリプトが含まれている: %s", body)
	}
}

// firstMeaningfulChildは、マークアップの体裁のための空白ではない最初の子を
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

// findCommentsはドキュメント内のHTMLコメントをすべて、ドキュメント順で
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
