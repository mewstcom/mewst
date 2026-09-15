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

// julyMonthは以下のフィクスチャが書き出す月。テストが渡す件数を宣言する。
func julyMonth(postCount int64) usecase.ExportArchiveMonth {
	return newMonth(2026, time.July, postCount)
}

// julyPostはアーカイブのゾーンにおける投稿日と時刻から、2026年7月の
// ポストを組み立てる。
func julyPost(t *testing.T, id string, day, hour, minute int, content string) usecase.ExportArchivePost {
	t.Helper()

	return newPost(id, time.Date(2026, time.July, day, hour, minute, 0, 0, mustLoadLocation(t, "Asia/Tokyo")), content)
}

// buildMonthEntryは1か月分だけを持つアーカイブを構築し、その月のエントリ
// をzipに格納された形で返す。契約はテンプレートの出力ではなく格納されたエントリ
// に対して検証する。読み手がアーカイブを解凍して開くのはそのエントリのため。
func buildMonthEntry(t *testing.T, posts ...usecase.ExportArchivePost) string {
	t.Helper()

	ctx := context.Background()
	month := julyMonth(int64(len(posts)))
	archive := newArchive(t, month)

	var buf bytes.Buffer
	writer := exportfile.NewBuilder().NewArchive(&buf, archive)
	if err := writer.WriteIndex(ctx); err != nil {
		t.Fatalf("index.htmlの書き出しに失敗: %v", err)
	}
	writeMonthEntry(t, ctx, writer, month, posts)
	if err := writer.Close(); err != nil {
		t.Fatalf("アーカイブのクローズに失敗: %v", err)
	}
	return entryBody(t, readArchive(t, buf.Bytes()), "posts/2026-07.html")
}

// postArticlesは月のエントリに含まれる各ポストのarticleを、ドキュメント順
// で返す。
func postArticles(t *testing.T, body string) []*html.Node {
	t.Helper()

	return findElements(parseHTML(t, body), "article")
}

// postContentTextはパーサーがポストの本文から復元するテキストコンテンツを
// 返す。アーカイブが元のまま返すと約束しているのはこれである。
func postContentText(t *testing.T, article *html.Node) string {
	t.Helper()

	for _, paragraph := range findElements(article, "p") {
		if strings.Contains(attribute(paragraph, "class"), "e-content") {
			return elementText(paragraph)
		}
	}
	t.Fatalf("ポストに .e-contentが無い")
	return ""
}

func TestMonthHTML_FollowsDocumentContract(t *testing.T) {
	t.Parallel()

	body := buildMonthEntry(t, julyPost(t, "post-1", 23, 21, 30, "7 月のポスト"))

	// 月のファイルは、ファイルアプリからも目次からも単体で開かれるため、
	// index.htmlと同じドキュメントの契約を満たす。
	assertDocumentContract(t, body, "ja")

	if title := elementText(findElement(t, parseHTML(t, body), "title")); !strings.Contains(title, "2026") {
		t.Errorf("title = %q、期待値 = 月を示す文言", title)
	}
	if heading := elementText(findElement(t, parseHTML(t, body), "h1")); !strings.Contains(heading, "2026") {
		t.Errorf("h1 = %q、期待値 = 月を示す文言", heading)
	}
}

func TestMonthHTML_DocumentsThePostContractInAComment(t *testing.T) {
	t.Parallel()

	body := buildMonthEntry(t, julyPost(t, "post-1", 23, 21, 30, "7 月のポスト"))

	document := parseHTML(t, body)
	comments := findComments(document)
	if len(comments) != 1 {
		t.Fatalf("コメントの数 = %d、期待値 = 1: %s", len(comments), body)
	}

	// コメントはbodyの先頭に置く。doctypeより前ではブラウザがquirks
	// モードへ落ち、それより後ろではエディタでファイルを開いた読み手が探す
	// ことになるため。
	if got := firstMeaningfulChild(findElement(t, document, "body")); got != comments[0] {
		t.Errorf("仕様コメントがbodyの最初の内容ではない: %s", body)
	}

	// コメントはマークアップが従う出力契約をそのまま示す。ファイルを開いた
	// 読み手が、そのファイルだけからパーサーを書けるようにするため。
	for _, want := range []string{"v1", "article.h-entry", "data-post-id", "time.dt-published", "datetime", ".e-content"} {
		if !strings.Contains(comments[0].Data, want) {
			t.Errorf("仕様コメントに%qが含まれていない: %q", want, comments[0].Data)
		}
	}
	// コメントは連続したハイフンを含むと途中で終わってしまい、そのテキストは
	// rawで書き出される。
	if strings.Contains(comments[0].Data, "--") {
		t.Errorf("仕様コメントにコメントを終端させる%qが含まれている: %q", "--", comments[0].Data)
	}
}

func TestMonthHTML_WritesEveryPostInTheOrderItIsGiven(t *testing.T) {
	t.Parallel()

	// ポストはpublished_at、次いでポストIDの順で渡されるため、同じ時点に
	// 投稿された2件はIDの順序を保つ。
	body := buildMonthEntry(t,
		julyPost(t, "post-a", 1, 9, 0, "1 件目"),
		julyPost(t, "post-b", 23, 21, 30, "2 件目"),
		julyPost(t, "post-c", 23, 21, 30, "3 件目"),
	)

	articles := postArticles(t, body)
	wantIDs := []string{"post-a", "post-b", "post-c"}
	if len(articles) != len(wantIDs) {
		t.Fatalf("ポストの数 = %d、期待値 = %d: %s", len(articles), len(wantIDs), body)
	}
	for i, wantID := range wantIDs {
		if got := attribute(articles[i], "data-post-id"); got != wantID {
			t.Errorf("%d件目のdata-post-id = %q、期待値 = %q", i, got, wantID)
		}
	}
}

func TestMonthHTML_RecordsEveryFieldOfAPost(t *testing.T) {
	t.Parallel()

	body := buildMonthEntry(t, julyPost(t, "01JZQ8P0000000000000000000", 23, 21, 30, "7 月のポスト"))

	article := postArticles(t, body)[0]
	if class := attribute(article, "class"); !strings.Contains(class, "h-entry") {
		t.Errorf("article[class] = %q、h-entryを含むことを期待", class)
	}
	if got := attribute(article, "data-post-id"); got != "01JZQ8P0000000000000000000" {
		t.Errorf("data-post-id = %q、期待値 = %q", got, "01JZQ8P0000000000000000000")
	}

	published := findElement(t, article, "time")
	if class := attribute(published, "class"); !strings.Contains(class, "dt-published") {
		t.Errorf("time[class] = %q、dt-publishedを含むことを期待", class)
	}

	// datetimeはアーカイブを描画したゾーンのオフセットを保つため、別の
	// 場所でファイルを読んでも時点が一意に定まる。その隣のテキストは、読み手が
	// 投稿した壁時計を示す。
	const wantDatetime = "2026-07-23T21:30:00+09:00"
	datetime := attribute(published, "datetime")
	if datetime != wantDatetime {
		t.Errorf("time[datetime] = %q、期待値 = %q", datetime, wantDatetime)
	}
	if _, err := time.Parse(time.RFC3339, datetime); err != nil {
		t.Errorf("time[datetime] がRFC 3339として解釈できない: %v", err)
	}
	if text := elementText(published); !strings.Contains(text, "21:30") {
		t.Errorf("投稿日時の可視表記 = %q、投稿時刻を含むことを期待", text)
	}

	if got := postContentText(t, article); got != "7 月のポスト" {
		t.Errorf(".e-contentのテキスト = %q、期待値 = %q", got, "7 月のポスト")
	}
}

func TestMonthHTML_EscapesPostContent(t *testing.T) {
	t.Parallel()

	// ポストの本文は書き手が入力したそのままであるため、その中のマークアップ
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
		t.Errorf(".e-contentのテキスト = %q、期待値 = %q", got, payload)
	}
	// IDは属性へ埋め込まれるため、その中の引用符が属性を閉じられては
	// ならない。
	if got := attribute(article, "data-post-id"); got != `post-"1"` {
		t.Errorf("data-post-id = %q、期待値 = %q", got, `post-"1"`)
	}
}

func TestMonthHTML_KeepsTheBodyAsItWasWritten(t *testing.T) {
	t.Parallel()

	// 改行はマークアップに変換せず改行のまま保つため、要素のテキスト
	// コンテンツから元の本文がそのまま返る。
	const content = "1 行目\n\n3 行目 😀 絵文字と漢字\ttab"
	body := buildMonthEntry(t, julyPost(t, "post-1", 23, 21, 30, content))

	if got := postContentText(t, postArticles(t, body)[0]); got != content {
		t.Errorf(".e-contentのテキスト = %q、期待値 = %q", got, content)
	}
	if strings.Contains(body, "<br") {
		t.Errorf("本文の改行が <br> に変換されている: %s", body)
	}
	// 改行が見えるのは、本文が格納されたとおりに表示される場合だけである。
	if !strings.Contains(body, "white-space: pre-wrap") {
		t.Errorf("スタイルに本文の改行を表示する指定が無い: %s", body)
	}
}

func TestMonthHTML_WritesAMonthWithoutPosts(t *testing.T) {
	t.Parallel()

	// 月が宣言されるのはポストを持つときだけだが、1件も無いエントリでも
	// 切り詰められたファイルではなく完結したドキュメントになる。
	body := buildMonthEntry(t)

	assertDocumentContract(t, body, "ja")
	if articles := postArticles(t, body); len(articles) != 0 {
		t.Errorf("ポストの数 = %d、期待値 = 0: %s", len(articles), body)
	}
}
