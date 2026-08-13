package exportfile_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/html"

	"github.com/mewstcom/mewst/go/internal/exportfile"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// julyMonth is the month the fixtures below write, with as many posts declared
// as the test hands over.
//
// [Ja] julyMonth は以下のフィクスチャが書き出す月。テストが渡す件数を宣言する。
func julyMonth(postCount int64) usecase.ExportArchiveMonth {
	return newMonth(2026, time.July, postCount)
}

// julyPost builds a post of July 2026 from the day and time of day it was
// published in the archive's zone.
//
// [Ja] julyPost はアーカイブのゾーンにおける投稿日と時刻から、2026 年 7 月の
// ポストを組み立てる。
func julyPost(t *testing.T, id string, day, hour, minute int, content string) usecase.ExportArchivePost {
	t.Helper()

	return newPost(id, time.Date(2026, time.July, day, hour, minute, 0, 0, mustLoadLocation(t, "Asia/Tokyo")), content)
}

// buildMonthEntry builds an archive holding a single month and returns that
// month's entry as it is stored in the zip. The contract is asserted on the
// stored entry rather than on the template's output, because that entry is
// what a reader opens after unpacking the archive.
//
// [Ja] buildMonthEntry は 1 か月分だけを持つアーカイブを構築し、その月のエントリ
// を zip に格納された形で返す。契約はテンプレートの出力ではなく格納されたエントリ
// に対して検証する。読み手がアーカイブを解凍して開くのはそのエントリのため。
func buildMonthEntry(t *testing.T, posts ...usecase.ExportArchivePost) string {
	t.Helper()

	ctx := context.Background()
	month := julyMonth(int64(len(posts)))
	archive := newArchive(t, month)

	var buf bytes.Buffer
	writer := exportfile.NewBuilder().NewArchive(&buf, archive)
	if err := writer.WriteIndex(ctx); err != nil {
		t.Fatalf("index.html の書き出しに失敗: %v", err)
	}
	writeMonthEntry(t, ctx, writer, month, posts)
	if err := writer.Close(); err != nil {
		t.Fatalf("アーカイブのクローズに失敗: %v", err)
	}
	return entryBody(t, readArchive(t, buf.Bytes()), "posts/2026-07.html")
}

// postArticles returns the article of every post of a month's entry, in
// document order.
//
// [Ja] postArticles は月のエントリに含まれる各ポストの article を、ドキュメント順
// で返す。
func postArticles(t *testing.T, body string) []*html.Node {
	t.Helper()

	return findElements(parseHTML(t, body), "article")
}

// postContentText returns the text content a parser recovers from a post's
// body, which is what the archive promises to give back unchanged.
//
// [Ja] postContentText はパーサーがポストの本文から復元するテキストコンテンツを
// 返す。アーカイブが元のまま返すと約束しているのはこれである。
func postContentText(t *testing.T, article *html.Node) string {
	t.Helper()

	for _, paragraph := range findElements(article, "p") {
		if strings.Contains(attribute(paragraph, "class"), "e-content") {
			return elementText(paragraph)
		}
	}
	t.Fatalf("ポストに .e-content が無い")
	return ""
}

func TestMonthHTML_FollowsDocumentContract(t *testing.T) {
	t.Parallel()

	body := buildMonthEntry(t, julyPost(t, "post-1", 23, 21, 30, "7 月のポスト"))

	// A month's file is opened on its own, straight from the file manager or
	// from the table of contents, so it carries the same document contract as
	// index.html.
	//
	// [Ja] 月のファイルは、ファイルアプリからも目次からも単体で開かれるため、
	// index.html と同じドキュメントの契約を満たす。
	assertDocumentContract(t, body, "ja")

	if title := elementText(findElement(t, parseHTML(t, body), "title")); !strings.Contains(title, "2026") {
		t.Errorf("title = %q, want 月を示す文言", title)
	}
	if heading := elementText(findElement(t, parseHTML(t, body), "h1")); !strings.Contains(heading, "2026") {
		t.Errorf("h1 = %q, want 月を示す文言", heading)
	}
}

func TestMonthHTML_DocumentsThePostContractInAComment(t *testing.T) {
	t.Parallel()

	body := buildMonthEntry(t, julyPost(t, "post-1", 23, 21, 30, "7 月のポスト"))

	document := parseHTML(t, body)
	comments := findComments(document)
	if len(comments) != 1 {
		t.Fatalf("コメントの数 = %d, want 1: %s", len(comments), body)
	}

	// The comment sits at the top of the body: before the doctype it would put
	// a browser into quirks mode, and anywhere further down a reader opening
	// the file in an editor would have to hunt for it.
	//
	// [Ja] コメントは body の先頭に置く。doctype より前ではブラウザが quirks
	// モードへ落ち、それより後ろではエディタでファイルを開いた読み手が探す
	// ことになるため。
	if got := firstMeaningfulChild(findElement(t, document, "body")); got != comments[0] {
		t.Errorf("仕様コメントが body の最初の内容ではない: %s", body)
	}

	// The comment names the same output contract the markup follows, so a
	// reader who opens the file can write a parser from the file alone.
	//
	// [Ja] コメントはマークアップが従う出力契約をそのまま示す。ファイルを開いた
	// 読み手が、そのファイルだけからパーサーを書けるようにするため。
	for _, want := range []string{"v1", "article.h-entry", "data-post-id", "time.dt-published", "datetime", ".e-content"} {
		if !strings.Contains(comments[0].Data, want) {
			t.Errorf("仕様コメントに %q が含まれていない: %q", want, comments[0].Data)
		}
	}
	// A comment cannot hold a run of hyphens without ending early, and the
	// text is written into it raw.
	//
	// [Ja] コメントは連続したハイフンを含むと途中で終わってしまい、そのテキストは
	// raw で書き出される。
	if strings.Contains(comments[0].Data, "--") {
		t.Errorf("仕様コメントにコメントを終端させる %q が含まれている: %q", "--", comments[0].Data)
	}
}

func TestMonthHTML_WritesEveryPostInTheOrderItIsGiven(t *testing.T) {
	t.Parallel()

	// The posts arrive ordered by published_at and then by post ID, so two
	// posts published at the same instant keep the order their IDs give them.
	//
	// [Ja] ポストは published_at、次いでポスト ID の順で渡されるため、同じ時点に
	// 投稿された 2 件は ID の順序を保つ。
	body := buildMonthEntry(t,
		julyPost(t, "post-a", 1, 9, 0, "1 件目"),
		julyPost(t, "post-b", 23, 21, 30, "2 件目"),
		julyPost(t, "post-c", 23, 21, 30, "3 件目"),
	)

	articles := postArticles(t, body)
	wantIDs := []string{"post-a", "post-b", "post-c"}
	if len(articles) != len(wantIDs) {
		t.Fatalf("ポストの数 = %d, want %d: %s", len(articles), len(wantIDs), body)
	}
	for i, wantID := range wantIDs {
		if got := attribute(articles[i], "data-post-id"); got != wantID {
			t.Errorf("%d 件目の data-post-id = %q, want %q", i, got, wantID)
		}
	}
}

func TestMonthHTML_RecordsEveryFieldOfAPost(t *testing.T) {
	t.Parallel()

	body := buildMonthEntry(t, julyPost(t, "01JZQ8P0000000000000000000", 23, 21, 30, "7 月のポスト"))

	article := postArticles(t, body)[0]
	if class := attribute(article, "class"); !strings.Contains(class, "h-entry") {
		t.Errorf("article[class] = %q, want h-entry を含む", class)
	}
	if got := attribute(article, "data-post-id"); got != "01JZQ8P0000000000000000000" {
		t.Errorf("data-post-id = %q, want %q", got, "01JZQ8P0000000000000000000")
	}

	published := findElement(t, article, "time")
	if class := attribute(published, "class"); !strings.Contains(class, "dt-published") {
		t.Errorf("time[class] = %q, want dt-published を含む", class)
	}

	// The datetime keeps the offset of the zone the archive was rendered in,
	// so the instant stays unambiguous when the file is read from elsewhere,
	// while the text next to it is the wall clock the reader posted at.
	//
	// [Ja] datetime はアーカイブを描画したゾーンのオフセットを保つため、別の
	// 場所でファイルを読んでも時点が一意に定まる。その隣のテキストは、読み手が
	// 投稿した壁時計を示す。
	const wantDatetime = "2026-07-23T21:30:00+09:00"
	datetime := attribute(published, "datetime")
	if datetime != wantDatetime {
		t.Errorf("time[datetime] = %q, want %q", datetime, wantDatetime)
	}
	if _, err := time.Parse(time.RFC3339, datetime); err != nil {
		t.Errorf("time[datetime] が RFC 3339 として解釈できない: %v", err)
	}
	if text := elementText(published); !strings.Contains(text, "21:30") {
		t.Errorf("投稿日時の可視表記 = %q, want 投稿時刻を含む", text)
	}

	if got := postContentText(t, article); got != "7 月のポスト" {
		t.Errorf(".e-content のテキスト = %q, want %q", got, "7 月のポスト")
	}
}

func TestMonthHTML_EscapesPostContent(t *testing.T) {
	t.Parallel()

	// A post body is whatever its author typed, so markup in it has to come
	// back out as the text it was, not as markup of the archive.
	//
	// [Ja] ポストの本文は書き手が入力したそのままであるため、その中のマークアップ
	// はアーカイブのマークアップではなく、元のテキストとして返る必要がある。
	const payload = `<script>alert("xss")</script><img src=x onerror=alert(1)>& "quoted" 'single'`
	body := buildMonthEntry(t, julyPost(t, `post-"1"`, 23, 21, 30, payload))

	document := parseHTML(t, body)
	if scripts := findElements(document, "script"); len(scripts) != 0 {
		t.Errorf("本文のマークアップが要素として解釈されている (script): %s", body)
	}
	if images := findElements(document, "img"); len(images) != 0 {
		t.Errorf("本文のマークアップが要素として解釈されている (img): %s", body)
	}

	article := postArticles(t, body)[0]
	if got := postContentText(t, article); got != payload {
		t.Errorf(".e-content のテキスト = %q, want %q", got, payload)
	}
	// An ID is interpolated into an attribute, so a quote in it must not be
	// able to close that attribute.
	//
	// [Ja] ID は属性へ埋め込まれるため、その中の引用符が属性を閉じられては
	// ならない。
	if got := attribute(article, "data-post-id"); got != `post-"1"` {
		t.Errorf("data-post-id = %q, want %q", got, `post-"1"`)
	}
}

func TestMonthHTML_KeepsTheBodyAsItWasWritten(t *testing.T) {
	t.Parallel()

	// Newlines are kept as newlines rather than turned into markup, so the
	// original body comes back out of the element's text content.
	//
	// [Ja] 改行はマークアップに変換せず改行のまま保つため、要素のテキスト
	// コンテンツから元の本文がそのまま返る。
	const content = "1 行目\n\n3 行目 😀 絵文字と漢字\ttab"
	body := buildMonthEntry(t, julyPost(t, "post-1", 23, 21, 30, content))

	if got := postContentText(t, postArticles(t, body)[0]); got != content {
		t.Errorf(".e-content のテキスト = %q, want %q", got, content)
	}
	if strings.Contains(body, "<br") {
		t.Errorf("本文の改行が <br> に変換されている: %s", body)
	}
	// The newlines are only visible if the body is displayed as it is stored.
	//
	// [Ja] 改行が見えるのは、本文が格納されたとおりに表示される場合だけである。
	if !strings.Contains(body, "white-space: pre-wrap") {
		t.Errorf("スタイルに本文の改行を表示する指定が無い: %s", body)
	}
}

func TestMonthHTML_WritesAMonthWithoutPosts(t *testing.T) {
	t.Parallel()

	// A month is only declared when it holds posts, but an entry with none is
	// still a complete document rather than a truncated file.
	//
	// [Ja] 月が宣言されるのはポストを持つときだけだが、1 件も無いエントリでも
	// 切り詰められたファイルではなく完結したドキュメントになる。
	body := buildMonthEntry(t)

	assertDocumentContract(t, body, "ja")
	if articles := postArticles(t, body); len(articles) != 0 {
		t.Errorf("ポストの数 = %d, want 0: %s", len(articles), body)
	}
}
