package exportfile_test

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/mewstcom/mewst/go/internal/exportfile"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// generatedAt is the archive-wide timestamp the fixtures stamp their entries
// with.
//
// [Ja] generatedAt はフィクスチャが各エントリに記録するアーカイブ共通の時刻。
var generatedAt = time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)

// mustLoadLocation resolves the zone the fixtures render their timestamps in.
//
// [Ja] mustLoadLocation はフィクスチャが日時を描画するゾーンを解決する。
func mustLoadLocation(t testing.TB, name string) *time.Location {
	t.Helper()

	location, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("タイムゾーンの解決に失敗: %v", err)
	}
	return location
}

// newMonth builds a declared month. LocalMonthStart is a calendar label, so it
// is written as the wall clock of the month's first day.
//
// [Ja] newMonth は宣言する月を組み立てる。LocalMonthStart は暦月のラベルのため、
// その月の初日の壁時計として書く。
func newMonth(year int, month time.Month, postCount int64) usecase.ExportArchiveMonth {
	return usecase.ExportArchiveMonth{
		LocalMonthStart: time.Date(year, month, 1, 0, 0, 0, 0, time.UTC),
		PostCount:       postCount,
	}
}

// newArchive builds an archive whose months are rendered in Asia/Tokyo.
//
// [Ja] newArchive は Asia/Tokyo で描画するアーカイブを組み立てる。
func newArchive(t testing.TB, months ...usecase.ExportArchiveMonth) usecase.ExportArchive {
	t.Helper()

	return usecase.ExportArchive{
		Locale:      "ja",
		Location:    mustLoadLocation(t, "Asia/Tokyo"),
		Months:      months,
		GeneratedAt: generatedAt,
	}
}

// newPost builds one post of an archive.
//
// [Ja] newPost はアーカイブに含める投稿を 1 件組み立てる。
func newPost(id string, publishedAt time.Time, content string) usecase.ExportArchivePost {
	return usecase.ExportArchivePost{ID: id, Content: content, PublishedAt: publishedAt}
}

// writeMonthEntry writes one month from start to finish.
//
// [Ja] writeMonthEntry は 1 か月分のエントリを最初から最後まで書き出す。
func writeMonthEntry(
	t *testing.T,
	ctx context.Context,
	writer usecase.ExportArchiveWriter,
	month usecase.ExportArchiveMonth,
	posts []usecase.ExportArchivePost,
) {
	t.Helper()

	monthWriter, err := writer.OpenMonth(ctx, month)
	if err != nil {
		t.Fatalf("月のエントリの作成に失敗: %v", err)
	}
	for _, post := range posts {
		if err := monthWriter.WritePost(ctx, post); err != nil {
			t.Fatalf("投稿の書き出しに失敗: %v", err)
		}
	}
	if err := monthWriter.Close(); err != nil {
		t.Fatalf("月のエントリのクローズに失敗: %v", err)
	}
}

// archiveEntry is one entry read back from a built archive.
//
// [Ja] archiveEntry は構築したアーカイブから読み戻したエントリ 1 つ。
type archiveEntry struct {
	name     string
	body     string
	modified time.Time
}

// readArchive reads every entry of a built archive in stored order.
//
// [Ja] readArchive は構築したアーカイブの全エントリを格納順に読み取る。
func readArchive(t *testing.T, data []byte) []archiveEntry {
	t.Helper()

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("アーカイブの読み取りに失敗: %v", err)
	}

	entries := make([]archiveEntry, 0, len(reader.File))
	for _, file := range reader.File {
		body := readEntry(t, file)
		entries = append(entries, archiveEntry{name: file.Name, body: body, modified: file.Modified})
	}
	return entries
}

// readEntry reads one entry's body.
//
// [Ja] readEntry はエントリ 1 つの本文を読み取る。
func readEntry(t *testing.T, file *zip.File) string {
	t.Helper()

	rc, err := file.Open()
	if err != nil {
		t.Fatalf("エントリのオープンに失敗 (entry: %s): %v", file.Name, err)
	}
	defer func() { _ = rc.Close() }()

	body, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("エントリの読み取りに失敗 (entry: %s): %v", file.Name, err)
	}
	return string(body)
}

// entryBody returns the body of the named entry.
//
// [Ja] entryBody は指定した名前のエントリの本文を返す。
func entryBody(t *testing.T, entries []archiveEntry, name string) string {
	t.Helper()

	for _, entry := range entries {
		if entry.name == name {
			return entry.body
		}
	}
	t.Fatalf("エントリが見つからない (entry: %s)", name)
	return ""
}

// countingWriter records how many bytes the archive has handed to the
// underlying writer so far.
//
// [Ja] countingWriter はアーカイブがここまでに下位 writer へ渡したバイト数を
// 記録する。
type countingWriter struct {
	written int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	c.written += int64(len(p))
	return len(p), nil
}

// errWriterStopped is the failure stoppableWriter returns once it is stopped.
//
// [Ja] errWriterStopped は stoppableWriter が停止した後に返す失敗。
var errWriterStopped = errors.New("書き込みに失敗")

// stoppableWriter accepts every write until it is stopped, then fails them all.
// The zip writer buffers its output, so a write only reaches this writer once
// the archive has produced enough bytes to flush that buffer. Tests that need a
// failure therefore have to hand the archive a realistic amount of data first.
//
// [Ja] stoppableWriter は停止させるまで全ての書き込みを受け付け、停止後は全て
// 失敗させる。zip writer は出力をバッファするため、この writer まで書き込みが
// 届くのは、アーカイブがそのバッファを流し出すだけのバイト数を生んだ後になる。
// そのため失敗を起こすテストは、先に相応の量のデータをアーカイブへ渡す必要が
// ある。
type stoppableWriter struct {
	stopped bool
}

func (s *stoppableWriter) Write(p []byte) (int, error) {
	if s.stopped {
		return 0, errWriterStopped
	}
	return len(p), nil
}

// newPosts builds count posts of one month, each long enough that the archive
// keeps handing compressed output to the writer underneath it.
//
// [Ja] newPosts は 1 か月分の投稿を count 件組み立てる。各投稿は、アーカイブが
// 下位 writer へ圧縮済みの出力を渡し続ける程度の長さを持つ。
func newPosts(month time.Month, count int) []usecase.ExportArchivePost {
	posts := make([]usecase.ExportArchivePost, 0, count)
	for i := range count {
		posts = append(posts, newPost(
			fmt.Sprintf("post-%d", i),
			time.Date(2026, month, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(i)*time.Minute),
			fmt.Sprintf("%d 件目のポスト %s", i, strings.Repeat("あ", 100)),
		))
	}
	return posts
}

func TestBuilder_WritesIndexAndMonthEntriesInOrder(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	june := newMonth(2026, time.June, 1)
	july := newMonth(2026, time.July, 2)
	archive := newArchive(t, june, july)

	var buf bytes.Buffer
	writer := exportfile.NewBuilder().NewArchive(&buf, archive)

	if err := writer.WriteIndex(ctx); err != nil {
		t.Fatalf("index.html の書き出しに失敗: %v", err)
	}
	writeMonthEntry(t, ctx, writer, june, []usecase.ExportArchivePost{
		newPost("post-june", time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), "6 月のポスト"),
	})
	writeMonthEntry(t, ctx, writer, july, []usecase.ExportArchivePost{
		// Published at 00:30 on July 1st in Asia/Tokyo, so a UTC rendering
		// would put it in the June entry's month.
		//
		// [Ja] Asia/Tokyo では 7 月 1 日 00:30 の公開のため、UTC で描画すると
		// 6 月のエントリの月になってしまう。
		newPost("post-july-1", time.Date(2026, 6, 30, 15, 30, 0, 0, time.UTC), "1 行目\n2 行目"),
		newPost("post-july-2", time.Date(2026, 7, 5, 2, 0, 0, 0, time.UTC), `<script>alert("x")</script>`),
	})
	if err := writer.Close(); err != nil {
		t.Fatalf("アーカイブのクローズに失敗: %v", err)
	}

	entries := readArchive(t, buf.Bytes())
	wantNames := []string{"index.html", "posts/2026-06.html", "posts/2026-07.html"}
	if len(entries) != len(wantNames) {
		t.Fatalf("エントリ数 = %d, want %d", len(entries), len(wantNames))
	}
	for i, wantName := range wantNames {
		if entries[i].name != wantName {
			t.Errorf("エントリ %d の名前 = %q, want %q", i, entries[i].name, wantName)
		}
		if got := entries[i].modified.UTC(); !got.Equal(generatedAt) {
			t.Errorf("エントリ %q の更新時刻 = %v, want %v", entries[i].name, got, generatedAt)
		}
	}

	// The table of contents links the entries the archive holds. What that
	// document looks like is fixed by the format contract in
	// index_html_test.go.
	//
	// [Ja] 目次はアーカイブが持つエントリへリンクする。その文書の形は
	// index_html_test.go の format 契約で固定している。
	index := entryBody(t, entries, "index.html")
	for _, want := range []string{
		`href="posts/2026-06.html"`,
		`href="posts/2026-07.html"`,
	} {
		if !strings.Contains(index, want) {
			t.Errorf("index.html に %q が含まれていない: %s", want, index)
		}
	}

	julyBody := entryBody(t, entries, "posts/2026-07.html")
	if !strings.HasPrefix(julyBody, "<!doctype html>") {
		t.Errorf("月のエントリが doctype で始まっていない: %s", julyBody)
	}
	if !strings.HasSuffix(julyBody, "</html>\n") {
		t.Errorf("月のエントリが閉じられていない: %s", julyBody)
	}
	for _, want := range []string{
		`data-post-id="post-july-1"`,
		`datetime="2026-07-01T00:30:00+09:00"`,
		"1 行目\n2 行目",
		"&lt;script&gt;",
	} {
		if !strings.Contains(julyBody, want) {
			t.Errorf("7 月のエントリに %q が含まれていない: %s", want, julyBody)
		}
	}
	if strings.Contains(julyBody, "<script>") {
		t.Errorf("投稿本文がエスケープされていない: %s", julyBody)
	}
	if strings.Contains(julyBody, "post-june") {
		t.Errorf("7 月のエントリに 6 月の投稿が含まれている: %s", julyBody)
	}
}

func TestBuilder_StreamsEntriesBeforeClose(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	const postCount = 5000
	july := newMonth(2026, time.July, postCount)
	archive := newArchive(t, july)

	counter := &countingWriter{}
	writer := exportfile.NewBuilder().NewArchive(counter, archive)

	if err := writer.WriteIndex(ctx); err != nil {
		t.Fatalf("index.html の書き出しに失敗: %v", err)
	}
	monthWriter, err := writer.OpenMonth(ctx, july)
	if err != nil {
		t.Fatalf("月のエントリの作成に失敗: %v", err)
	}
	afterOpen := counter.written

	for _, post := range newPosts(time.July, postCount) {
		if err := monthWriter.WritePost(ctx, post); err != nil {
			t.Fatalf("投稿の書き出しに失敗: %v", err)
		}
	}

	// Bytes for the posts must reach the writer while the month is still open.
	// If the builder buffered the month, the count would not have increased
	// since OpenMonth wrote the entry header.
	//
	// [Ja] 投稿のバイト列は月のエントリが開いている間に writer へ届く必要がある。
	// builder が月全体をバッファしていれば、OpenMonth がエントリヘッダーを
	// 書き出した後からバイト数は増えない。
	beforeClose := counter.written
	if beforeClose <= afterOpen {
		t.Fatalf(
			"投稿の書き出し後のバイト数 = %d, want > OpenMonth 直後の %d",
			beforeClose,
			afterOpen,
		)
	}

	if err := monthWriter.Close(); err != nil {
		t.Fatalf("月のエントリのクローズに失敗: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("アーカイブのクローズに失敗: %v", err)
	}
	if counter.written <= beforeClose {
		t.Errorf("クローズ後の書き出しバイト数 = %d, want > %d", counter.written, beforeClose)
	}
}

func TestBuilder_StopsWhenReaderCloses(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	const postCount = 20000
	july := newMonth(2026, time.July, postCount)
	archive := newArchive(t, july)

	reader, pipeWriter := io.Pipe()
	uploadErr := errors.New("アップロードに失敗")
	done := make(chan error, 1)

	go func() {
		writer := exportfile.NewBuilder().NewArchive(pipeWriter, archive)
		done <- buildArchive(ctx, writer, july, postCount)
	}()

	// Consume enough to leave the builder mid-archive, then fail the read side
	// the way a failed upload does.
	//
	// [Ja] builder がアーカイブの途中で止まる程度まで読み進めてから、失敗した
	// アップロードと同じように読み取り側を失敗させる。
	if _, err := io.ReadFull(reader, make([]byte, 512)); err != nil {
		t.Fatalf("アーカイブの読み取りに失敗: %v", err)
	}
	if err := reader.CloseWithError(uploadErr); err != nil {
		t.Fatalf("パイプのクローズに失敗: %v", err)
	}

	select {
	case err := <-done:
		if !errors.Is(err, uploadErr) {
			t.Errorf("builder のエラー = %v, want %v", err, uploadErr)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("読み取り側を閉じても builder が終了しない")
	}
}

// buildArchive writes a whole archive and returns the first error, mirroring
// how the generation use case drives the builder.
//
// [Ja] buildArchive はアーカイブ全体を書き出し、最初のエラーを返す。生成の
// UseCase が builder を駆動する形を模している。
func buildArchive(
	ctx context.Context,
	writer usecase.ExportArchiveWriter,
	month usecase.ExportArchiveMonth,
	postCount int,
) error {
	if err := writer.WriteIndex(ctx); err != nil {
		return err
	}

	monthWriter, err := writer.OpenMonth(ctx, month)
	if err != nil {
		return err
	}
	for _, post := range newPosts(month.LocalMonthStart.Month(), postCount) {
		if err := monthWriter.WritePost(ctx, post); err != nil {
			return err
		}
	}
	if err := monthWriter.Close(); err != nil {
		return err
	}
	return writer.Close()
}

func TestBuilder_CloseFinishesOpenMonthEntry(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	july := newMonth(2026, time.July, 1)
	archive := newArchive(t, july)

	var buf bytes.Buffer
	writer := exportfile.NewBuilder().NewArchive(&buf, archive)

	if err := writer.WriteIndex(ctx); err != nil {
		t.Fatalf("index.html の書き出しに失敗: %v", err)
	}
	monthWriter, err := writer.OpenMonth(ctx, july)
	if err != nil {
		t.Fatalf("月のエントリの作成に失敗: %v", err)
	}
	post := newPost("post-july", time.Date(2026, 7, 5, 2, 0, 0, 0, time.UTC), "7 月のポスト")
	if err := monthWriter.WritePost(ctx, post); err != nil {
		t.Fatalf("投稿の書き出しに失敗: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("アーカイブのクローズに失敗: %v", err)
	}

	julyBody := entryBody(t, readArchive(t, buf.Bytes()), "posts/2026-07.html")
	if !strings.HasSuffix(julyBody, "</html>\n") {
		t.Errorf("開いたままの月のエントリが閉じられていない: %s", julyBody)
	}
}

func TestBuilder_CloseRejectsIncompleteArchive(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	july := newMonth(2026, time.July, 1)

	t.Run("index.html が書き出されていない", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		writer := exportfile.NewBuilder().NewArchive(&buf, newArchive(t))
		if err := writer.Close(); err == nil {
			t.Fatal("index.html が無いアーカイブの Close がエラーにならない")
		}

		if entries := readArchive(t, buf.Bytes()); len(entries) != 0 {
			t.Errorf("不完全なアーカイブのエントリ数 = %d, want 0", len(entries))
		}
	})

	t.Run("宣言した月が書き出されていない", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		writer := exportfile.NewBuilder().NewArchive(&buf, newArchive(t, july))
		if err := writer.WriteIndex(ctx); err != nil {
			t.Fatalf("index.html の書き出しに失敗: %v", err)
		}
		if err := writer.Close(); err == nil {
			t.Fatal("宣言した月が無いアーカイブの Close がエラーにならない")
		}

		entries := readArchive(t, buf.Bytes())
		if len(entries) != 1 || entries[0].name != "index.html" {
			t.Errorf("不完全なアーカイブのエントリ = %+v, want index.html のみ", entries)
		}
	})

	t.Run("同じ月が重複して宣言されている", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		writer := exportfile.NewBuilder().NewArchive(&buf, newArchive(t, july, july))
		if err := writer.WriteIndex(ctx); err == nil {
			t.Error("月が重複したアーカイブの WriteIndex がエラーにならない")
		}
		if err := writer.Close(); err == nil {
			t.Error("月が重複したアーカイブの Close がエラーにならない")
		}

		if entries := readArchive(t, buf.Bytes()); len(entries) != 0 {
			t.Errorf("不完全なアーカイブのエントリ数 = %d, want 0", len(entries))
		}
	})
}

func TestBuilder_KeepsIndexWriteFailureUntilClose(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	// The table of contents has to be large enough that writing it reaches the
	// writer underneath the zip writer's buffer. A small index is still buffered
	// when WriteIndex returns, so it cannot fail at all.
	//
	// [Ja] 目次は、その書き出しが zip writer のバッファの先にある writer まで
	// 届く大きさである必要がある。小さな index は WriteIndex が返る時点でまだ
	// バッファに留まっているため、そもそも失敗しない。
	const monthCount = 6000
	months := make([]usecase.ExportArchiveMonth, 0, monthCount)
	start := time.Date(1526, time.January, 1, 0, 0, 0, 0, time.UTC)
	for i := range monthCount {
		months = append(months, usecase.ExportArchiveMonth{
			LocalMonthStart: start.AddDate(0, i, 0),
			PostCount:       int64(i),
		})
	}

	writer := exportfile.NewBuilder().NewArchive(&stoppableWriter{stopped: true}, newArchive(t, months...))

	if err := writer.WriteIndex(ctx); !errors.Is(err, errWriterStopped) {
		t.Fatalf("停止した writer での WriteIndex のエラー = %v, want %v", err, errWriterStopped)
	}

	// Close names the truncated index itself, so an incomplete archive is
	// detected by the builder's own state rather than by the underlying writer
	// still returning its error.
	//
	// [Ja] Close は切り詰められた index を自身で名指しする。不完全なアーカイブを、
	// 下位 writer がエラーを返し続けることではなく builder 自身の状態で検出する
	// ため。
	err := writer.Close()
	if want := "index.html の書き出しに失敗"; err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Close のエラー = %v, want %q を含む", err, want)
	}
}

func TestBuilder_KeepsMonthPendingWhenOpenMonthFails(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	june := newMonth(2026, time.June, 1000)
	july := newMonth(2026, time.July, 1)

	sink := &stoppableWriter{}
	writer := exportfile.NewBuilder().NewArchive(sink, newArchive(t, june, july))

	if err := writer.WriteIndex(ctx); err != nil {
		t.Fatalf("index.html の書き出しに失敗: %v", err)
	}
	// June's posts leave compressed output that opening July has to flush, so
	// the stopped writer is reached while OpenMonth is still creating July's
	// entry, before its header could be written.
	//
	// [Ja] 6 月の投稿は、7 月を開くときに流し出す必要がある圧縮出力を残す。
	// そのため OpenMonth が 7 月のエントリを作っている間、ヘッダーを書き出す
	// より前に、停止した writer まで書き込みが届く。
	writeMonthEntry(t, ctx, writer, june, newPosts(time.June, 1000))

	sink.stopped = true

	if _, err := writer.OpenMonth(ctx, july); !errors.Is(err, errWriterStopped) {
		t.Fatalf("停止した writer での OpenMonth のエラー = %v, want %v", err, errWriterStopped)
	}

	// A month leaves the pending set only after its entry is created and its
	// header is written, so a month whose OpenMonth failed is still reported as
	// missing instead of being taken for written.
	//
	// [Ja] 月が未処理の集合から外れるのは、エントリを作りヘッダーを書き出した
	// 後だけである。そのため OpenMonth が失敗した月は、書き出し済みと見なされず
	// 未出力として報告される。
	err := writer.Close()
	if err == nil {
		t.Fatal("月を書き出せなかったアーカイブの Close がエラーにならない")
	}
	if want := "目次に対応する月のエントリが書き出されていない (1 件)"; !strings.Contains(err.Error(), want) {
		t.Errorf("Close のエラー = %v, want %q を含む", err, want)
	}
}

func TestBuilder_CloseFailsAfterWritePostFailure(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	const postCount = 5000
	july := newMonth(2026, time.July, postCount)

	sink := &stoppableWriter{}
	writer := exportfile.NewBuilder().NewArchive(sink, newArchive(t, july))

	if err := writer.WriteIndex(ctx); err != nil {
		t.Fatalf("index.html の書き出しに失敗: %v", err)
	}
	monthWriter, err := writer.OpenMonth(ctx, july)
	if err != nil {
		t.Fatalf("月のエントリの作成に失敗: %v", err)
	}

	sink.stopped = true

	var writeErr error
	for _, post := range newPosts(time.July, postCount) {
		if writeErr = monthWriter.WritePost(ctx, post); writeErr != nil {
			break
		}
	}
	if !errors.Is(writeErr, errWriterStopped) {
		t.Fatalf("停止した writer での WritePost のエラー = %v, want %v", writeErr, errWriterStopped)
	}

	// The month is left open on purpose: a caller that saw the error only on
	// WritePost must not be able to finish the archive, so Close carries the
	// write failure even though the month was never closed by the caller.
	//
	// [Ja] 月は意図的に開いたままにする。WritePost でしかエラーを見ていない
	// 呼び出し側がアーカイブを完成させられないよう、呼び出し側が月を閉じて
	// いなくても Close は書き込みの失敗を持ち越す。
	if err := writer.Close(); !errors.Is(err, errWriterStopped) {
		t.Errorf("アーカイブの Close のエラー = %v, want %v を含む", err, errWriterStopped)
	}
}

func TestBuilder_CloseIsIdempotent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	july := newMonth(2026, time.July, 0)

	writer := exportfile.NewBuilder().NewArchive(&bytes.Buffer{}, newArchive(t, july))
	if err := writer.WriteIndex(ctx); err != nil {
		t.Fatalf("index.html の書き出しに失敗: %v", err)
	}
	writeMonthEntry(t, ctx, writer, july, nil)

	if err := writer.Close(); err != nil {
		t.Fatalf("アーカイブのクローズに失敗: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Errorf("2 回目の Close のエラー = %v, want nil", err)
	}
}

func TestBuilder_VerifiesDeclaredPostCount(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	post := newPost("post-july", time.Date(2026, 7, 5, 2, 0, 0, 0, time.UTC), "7 月のポスト")
	tests := []struct {
		name                string
		declaredPostCount   int64
		writes              int
		wantWriteErr        bool
		wantMonthCloseErr   bool
		wantArchiveCloseErr bool
	}{
		{name: "宣言件数どおり", declaredPostCount: 1, writes: 1},
		{name: "0 件", declaredPostCount: 0},
		{name: "宣言件数より少ない", declaredPostCount: 1, wantMonthCloseErr: true, wantArchiveCloseErr: true},
		{name: "宣言件数より多い", declaredPostCount: 1, writes: 2, wantWriteErr: true, wantMonthCloseErr: true, wantArchiveCloseErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			july := newMonth(2026, time.July, tt.declaredPostCount)
			writer := exportfile.NewBuilder().NewArchive(&bytes.Buffer{}, newArchive(t, july))
			if err := writer.WriteIndex(ctx); err != nil {
				t.Fatalf("index.html の書き出しに失敗: %v", err)
			}
			monthWriter, err := writer.OpenMonth(ctx, july)
			if err != nil {
				t.Fatalf("月のエントリの作成に失敗: %v", err)
			}

			var writeErr error
			for range tt.writes {
				if err := monthWriter.WritePost(ctx, post); err != nil {
					writeErr = err
					break
				}
			}
			if got := writeErr != nil; got != tt.wantWriteErr {
				t.Errorf("WritePost のエラー有無 = %t, want %t (error: %v)", got, tt.wantWriteErr, writeErr)
			}

			monthCloseErr := monthWriter.Close()
			if got := monthCloseErr != nil; got != tt.wantMonthCloseErr {
				t.Errorf("月の Close のエラー有無 = %t, want %t (error: %v)", got, tt.wantMonthCloseErr, monthCloseErr)
			}
			archiveCloseErr := writer.Close()
			if got := archiveCloseErr != nil; got != tt.wantArchiveCloseErr {
				t.Errorf("アーカイブの Close のエラー有無 = %t, want %t (error: %v)", got, tt.wantArchiveCloseErr, archiveCloseErr)
			}
		})
	}
}

func TestBuilder_RendersNilLocationAsUTC(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	july := newMonth(2026, time.July, 1)
	archive := usecase.ExportArchive{
		Locale:      "ja",
		Months:      []usecase.ExportArchiveMonth{july},
		GeneratedAt: generatedAt,
	}

	var buf bytes.Buffer
	writer := exportfile.NewBuilder().NewArchive(&buf, archive)

	if err := writer.WriteIndex(ctx); err != nil {
		t.Fatalf("index.html の書き出しに失敗: %v", err)
	}
	writeMonthEntry(t, ctx, writer, july, []usecase.ExportArchivePost{
		newPost("post-july", time.Date(2026, 7, 5, 2, 0, 0, 0, time.UTC), "7 月のポスト"),
	})
	if err := writer.Close(); err != nil {
		t.Fatalf("アーカイブのクローズに失敗: %v", err)
	}

	julyBody := entryBody(t, readArchive(t, buf.Bytes()), "posts/2026-07.html")
	if want := `datetime="2026-07-05T02:00:00Z"`; !strings.Contains(julyBody, want) {
		t.Errorf("nil の location のエントリに %q が含まれていない: %s", want, julyBody)
	}
}

func TestBuilder_CanceledContextStopsWriting(t *testing.T) {
	t.Parallel()

	july := newMonth(2026, time.July, 1)

	t.Run("WriteIndex", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		writer := exportfile.NewBuilder().NewArchive(&bytes.Buffer{}, newArchive(t, july))
		if err := writer.WriteIndex(ctx); !errors.Is(err, context.Canceled) {
			t.Errorf("キャンセル後の WriteIndex のエラー = %v, want %v", err, context.Canceled)
		}
	})

	t.Run("OpenMonth", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		writer := exportfile.NewBuilder().NewArchive(&bytes.Buffer{}, newArchive(t, july))
		if err := writer.WriteIndex(ctx); err != nil {
			t.Fatalf("index.html の書き出しに失敗: %v", err)
		}

		cancel()

		if _, err := writer.OpenMonth(ctx, july); !errors.Is(err, context.Canceled) {
			t.Errorf("キャンセル後の OpenMonth のエラー = %v, want %v", err, context.Canceled)
		}
	})

	t.Run("WritePost", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		writer := exportfile.NewBuilder().NewArchive(&bytes.Buffer{}, newArchive(t, july))
		if err := writer.WriteIndex(ctx); err != nil {
			t.Fatalf("index.html の書き出しに失敗: %v", err)
		}
		monthWriter, err := writer.OpenMonth(ctx, july)
		if err != nil {
			t.Fatalf("月のエントリの作成に失敗: %v", err)
		}

		cancel()

		post := newPost("post-july", time.Date(2026, 7, 5, 2, 0, 0, 0, time.UTC), "7 月のポスト")
		if err := monthWriter.WritePost(ctx, post); !errors.Is(err, context.Canceled) {
			t.Errorf("キャンセル後の WritePost のエラー = %v, want %v", err, context.Canceled)
		}
	})
}

func TestBuilder_RejectsInvalidSequence(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	june := newMonth(2026, time.June, 1)
	july := newMonth(2026, time.July, 1)
	august := newMonth(2026, time.August, 1)

	t.Run("index.html より前に月を開けない", func(t *testing.T) {
		t.Parallel()

		writer := exportfile.NewBuilder().NewArchive(&bytes.Buffer{}, newArchive(t, july))
		if _, err := writer.OpenMonth(ctx, july); err == nil {
			t.Error("index.html より前の OpenMonth がエラーにならない")
		}
	})

	t.Run("index.html を 2 回書き出せない", func(t *testing.T) {
		t.Parallel()

		writer := exportfile.NewBuilder().NewArchive(&bytes.Buffer{}, newArchive(t, july))
		if err := writer.WriteIndex(ctx); err != nil {
			t.Fatalf("index.html の書き出しに失敗: %v", err)
		}
		if err := writer.WriteIndex(ctx); err == nil {
			t.Error("2 回目の WriteIndex がエラーにならない")
		}
	})

	t.Run("目次に無い月は開けない", func(t *testing.T) {
		t.Parallel()

		writer := exportfile.NewBuilder().NewArchive(&bytes.Buffer{}, newArchive(t, july))
		if err := writer.WriteIndex(ctx); err != nil {
			t.Fatalf("index.html の書き出しに失敗: %v", err)
		}
		if _, err := writer.OpenMonth(ctx, august); err == nil {
			t.Error("宣言されていない月の OpenMonth がエラーにならない")
		}
	})

	t.Run("同じ月を 2 回開けない", func(t *testing.T) {
		t.Parallel()

		emptyJuly := newMonth(2026, time.July, 0)
		writer := exportfile.NewBuilder().NewArchive(&bytes.Buffer{}, newArchive(t, emptyJuly))
		if err := writer.WriteIndex(ctx); err != nil {
			t.Fatalf("index.html の書き出しに失敗: %v", err)
		}
		writeMonthEntry(t, ctx, writer, emptyJuly, nil)
		if _, err := writer.OpenMonth(ctx, emptyJuly); err == nil {
			t.Error("2 回目の OpenMonth がエラーにならない")
		}
	})

	t.Run("前の月を閉じずに次の月を開けない", func(t *testing.T) {
		t.Parallel()

		writer := exportfile.NewBuilder().NewArchive(&bytes.Buffer{}, newArchive(t, june, july))
		if err := writer.WriteIndex(ctx); err != nil {
			t.Fatalf("index.html の書き出しに失敗: %v", err)
		}
		if _, err := writer.OpenMonth(ctx, june); err != nil {
			t.Fatalf("月のエントリの作成に失敗: %v", err)
		}
		if _, err := writer.OpenMonth(ctx, july); err == nil {
			t.Error("前の月を閉じない OpenMonth がエラーにならない")
		}
	})

	t.Run("閉じた月には追記できない", func(t *testing.T) {
		t.Parallel()

		emptyJuly := newMonth(2026, time.July, 0)
		writer := exportfile.NewBuilder().NewArchive(&bytes.Buffer{}, newArchive(t, emptyJuly))
		if err := writer.WriteIndex(ctx); err != nil {
			t.Fatalf("index.html の書き出しに失敗: %v", err)
		}
		monthWriter, err := writer.OpenMonth(ctx, emptyJuly)
		if err != nil {
			t.Fatalf("月のエントリの作成に失敗: %v", err)
		}
		if err := monthWriter.Close(); err != nil {
			t.Fatalf("月のエントリのクローズに失敗: %v", err)
		}

		post := newPost("post-july", time.Date(2026, 7, 5, 2, 0, 0, 0, time.UTC), "7 月のポスト")
		if err := monthWriter.WritePost(ctx, post); err == nil {
			t.Error("閉じた月への WritePost がエラーにならない")
		}
	})
}
