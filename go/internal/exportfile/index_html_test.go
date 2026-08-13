package exportfile_test

import (
	"bytes"
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/html"

	"github.com/mewstcom/mewst/go/internal/exportfile"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// buildArchiveEntries builds a complete archive from its declared months,
// writing as many posts into each month as that month declares.
//
// [Ja] buildArchiveEntries は宣言された月からアーカイブを構築する。各月には、
// その月が宣言した件数の投稿を書き出す。
func buildArchiveEntries(t *testing.T, archive usecase.ExportArchive) []archiveEntry {
	t.Helper()

	ctx := context.Background()
	var buf bytes.Buffer
	writer := exportfile.NewBuilder().NewArchive(&buf, archive)
	if err := writer.WriteIndex(ctx); err != nil {
		t.Fatalf("index.html の書き出しに失敗: %v", err)
	}
	for _, month := range archive.Months {
		writeMonthEntry(t, ctx, writer, month, newPosts(month.LocalMonthStart.Month(), int(month.PostCount)))
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("アーカイブのクローズに失敗: %v", err)
	}
	return readArchive(t, buf.Bytes())
}

// buildIndex returns index.html as it is stored in the zip. The contract is
// asserted on the stored entry rather than on the template's output, because
// that entry is what a reader opens after unpacking the archive.
//
// [Ja] buildIndex は zip に格納された index.html を返す。契約はテンプレートの
// 出力ではなく格納されたエントリに対して検証する。読み手がアーカイブを解凍して
// 開くのはそのエントリのため。
func buildIndex(t *testing.T, archive usecase.ExportArchive) string {
	t.Helper()

	return entryBody(t, buildArchiveEntries(t, archive), "index.html")
}

// parseHTML parses an entry the way a browser would.
//
// [Ja] parseHTML はエントリをブラウザと同じように解析する。
func parseHTML(t *testing.T, body string) *html.Node {
	t.Helper()

	document, err := html.Parse(strings.NewReader(body))
	if err != nil {
		t.Fatalf("HTML の解析に失敗: %v", err)
	}
	return document
}

// findElements returns every element with the given tag, in document order.
//
// [Ja] findElements は指定したタグの要素をすべて、ドキュメント順で返す。
func findElements(node *html.Node, tag string) []*html.Node {
	var found []*html.Node
	if node.Type == html.ElementNode && node.Data == tag {
		found = append(found, node)
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		found = append(found, findElements(child, tag)...)
	}
	return found
}

// findElement returns the first element with the given tag.
//
// [Ja] findElement は指定したタグの最初の要素を返す。
func findElement(t *testing.T, node *html.Node, tag string) *html.Node {
	t.Helper()

	found := findElements(node, tag)
	if len(found) == 0 {
		t.Fatalf("要素が見つからない (tag: %s)", tag)
	}
	return found[0]
}

// attribute returns an element's attribute value.
//
// [Ja] attribute は要素の属性値を返す。
func attribute(node *html.Node, name string) string {
	for _, attr := range node.Attr {
		if attr.Key == name {
			return attr.Val
		}
	}
	return ""
}

// elementText returns an element's text content, which is what a reader sees
// and what a parser recovers from it.
//
// [Ja] elementText は要素のテキストコンテンツを返す。読み手が目にし、パーサーが
// 復元するのはこれである。
func elementText(node *html.Node) string {
	var text strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			text.WriteString(n.Data)
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return text.String()
}

func TestIndexHTML_FollowsDocumentContract(t *testing.T) {
	t.Parallel()

	assertDocumentContract(t, buildIndex(t, newArchive(t, newMonth(2026, time.June, 1), newMonth(2026, time.July, 12))), "ja")
}

func TestIndexHTML_IsSelfContainedAndFluid(t *testing.T) {
	t.Parallel()

	body := buildIndex(t, newArchive(t, newMonth(2026, time.July, 1)))

	// The archive is opened offline from a file manager, so nothing in the
	// document shell may be loaded from somewhere else. The month files share
	// that shell, but their post bodies can hold any text, so this is asserted
	// on the index.
	//
	// [Ja] アーカイブはファイルアプリからオフラインで開かれるため、ドキュメントの
	// 外枠は何ひとつ外部から読み込んではならない。月のファイルもその外枠を共有
	// するが、そちらは本文に任意のテキストが入りうるため、この検証は index に
	// 対して行う。
	for _, unwanted := range []string{"<script", "javascript:", "http://", "https://", "//fonts"} {
		if strings.Contains(body, unwanted) {
			t.Errorf("index.html に %q が含まれている: %s", unwanted, body)
		}
	}

	// A width in absolute units overflows a narrow screen and forces horizontal
	// scrolling, so the layout is sized in relative units only.
	//
	// [Ja] 絶対単位の幅は狭い画面からはみ出して横スクロールを強いるため、
	// レイアウトの寸法は相対単位だけで指定する。
	fixedWidth := regexp.MustCompile(`width\s*:\s*\d+\s*(px|pt|cm|mm|in|pc)`)
	if match := fixedWidth.FindString(body); match != "" {
		t.Errorf("スタイルに固定幅 %q が含まれている: %s", match, body)
	}
}

func TestIndexHTML_LinksEveryMonthEntryInOrder(t *testing.T) {
	t.Parallel()

	entries := buildArchiveEntries(t, newArchive(t, newMonth(2026, time.June, 1), newMonth(2026, time.July, 12)))
	storedNames := make(map[string]bool, len(entries))
	for _, entry := range entries {
		storedNames[entry.name] = true
	}

	links := findElements(parseHTML(t, entryBody(t, entries, "index.html")), "a")
	wantHrefs := []string{"posts/2026-06.html", "posts/2026-07.html"}
	if len(links) != len(wantHrefs) {
		t.Fatalf("目次のリンク数 = %d, want %d", len(links), len(wantHrefs))
	}
	for i, wantHref := range wantHrefs {
		href := attribute(links[i], "href")
		if href != wantHref {
			t.Errorf("リンク %d の href = %q, want %q", i, href, wantHref)
		}
		// A link to an entry the archive does not hold is a broken archive, and
		// the reader only finds out after unpacking it.
		//
		// [Ja] アーカイブが持たないエントリへのリンクは壊れたアーカイブであり、
		// 読み手はそれを解凍して初めて気づく。
		if !storedNames[href] {
			t.Errorf("目次がアーカイブに無いエントリへリンクしている (href: %s)", href)
		}
		// The link says which month it opens, so it still tells the reader where
		// it goes when a screen reader lists the links on their own.
		//
		// [Ja] リンクは開く月を示す。スクリーンリーダーがリンクだけを一覧しても
		// 行き先が分かるようにするため。
		if text := elementText(links[i]); !strings.Contains(text, "2026") {
			t.Errorf("リンク %d のテキスト = %q, want 月を示す文言", i, text)
		}
	}
}

func TestIndexHTML_RendersEmptyArchive(t *testing.T) {
	t.Parallel()

	body := buildIndex(t, newArchive(t))

	document := parseHTML(t, body)
	if lists := findElements(document, "ul"); len(lists) != 0 {
		t.Errorf("ポストが無いアーカイブの目次にリストがある: %s", body)
	}
	// A profile with no posts still gets an archive, so its table of contents
	// says that it is empty rather than leaving the reader with a bare heading.
	//
	// [Ja] 投稿が 1 件も無いプロフィールにもアーカイブは作られるため、その目次は
	// 見出しだけを残さず、空であることを伝える。
	main := findElement(t, document, "main")
	if paragraphs := findElements(main, "p"); len(paragraphs) != 1 || elementText(paragraphs[0]) == "" {
		t.Errorf("ポストが無いアーカイブの目次に説明が無い: %s", body)
	}
}

func TestIndexHTML_FallsBackToDefaultLanguage(t *testing.T) {
	t.Parallel()

	// An unknown locale resolves to the default language for the translations,
	// so the lang attribute has to name that same language.
	//
	// [Ja] 未知のロケールは翻訳では既定の言語に解決されるため、lang 属性も同じ
	// 言語を指す必要がある。
	archive := newArchive(t, newMonth(2026, time.July, 1))
	archive.Locale = "fr"

	if got := attribute(findElement(t, parseHTML(t, buildIndex(t, archive)), "html"), "lang"); got != "ja" {
		t.Errorf("未知のロケールの html[lang] = %q, want %q", got, "ja")
	}
}
