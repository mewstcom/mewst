package exportfile_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"runtime"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/mewstcom/mewst/go/internal/exportfile"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

const (
	// benchPostContentRunes is the post length the export requirements are
	// stated in, and benchArchiveMonths spreads the posts over three years so
	// that a profile with a long history is measured, not one huge month.
	//
	// [Ja] benchPostContentRunes はエクスポートの要件が前提とする投稿の長さで、
	// benchArchiveMonths は投稿を 3 年に分散させる。巨大な 1 か月ではなく、利用歴の
	// 長いプロフィールを計測するため。
	benchPostContentRunes = 160
	benchPostContentBytes = 480
	benchArchiveMonths    = 36

	// benchMaxHeapBytes is the added-heap target the generation flow is required
	// to stay under while it streams an archive.
	//
	// [Ja] benchMaxHeapBytes はアーカイブをストリーミングする生成処理が下回ることを
	// 求められる追加ヒープの目標値。
	benchMaxHeapBytes = 64 << 20
)

// benchFiller is the text posts take their content from. Posts cut different
// windows out of it, so the deflate stream sees text that repeats the way real
// posts do without every post being identical.
//
// [Ja] benchFiller は投稿の本文の取り出し元になる文。各投稿は別々の窓を切り出す
// ため、deflate は全投稿が同一になることなく、実際の投稿と同じように繰り返しを
// 含むテキストを受け取る。
var benchFiller = newBenchFiller()

// newBenchFiller builds the filler from a fixed seed, so that every run of the
// benchmark compresses exactly the same bytes.
//
// [Ja] newBenchFiller は固定シードから filler を組み立てる。ベンチマークの各実行が
// まったく同じバイト列を圧縮するようにするため。
func newBenchFiller() []rune {
	alphabet := []rune("あいうえおかきくけこさしすせそたちつてとなにぬねのはひふへほまみむめもやゆよらりるれろわをんアイウエオカキクケコサシスセソタチツテトナニヌネノ")
	random := rand.New(rand.NewPCG(1, 2))

	filler := make([]rune, 1<<18)
	for i := range filler {
		filler[i] = alphabet[random.IntN(len(alphabet))]
	}
	return filler
}

// benchArchiveCase is one shape of archive to measure. postCount separates the
// size of the export from pageSize, the number of posts one page of the
// generation loop holds at a time.
//
// [Ja] benchArchiveCase は計測するアーカイブの形。postCount はエクスポートの規模、
// pageSize は生成ループが一度に保持する投稿の件数で、両者を分けて指定する。
type benchArchiveCase struct {
	postCount int64
	pageSize  int64
}

// BenchmarkBuilder_StreamsArchive measures the archive generation of a profile
// with 100,000 posts. It reports the peak heap the streaming build adds, so the
// added heap can be checked against the 64 MiB target and, by comparing the
// cases, shown to follow the page size rather than the number of posts.
//
// [Ja] BenchmarkBuilder_StreamsArchive は 10 万件の投稿を持つプロフィールの
// アーカイブ生成を計測する。ストリーミング生成が追加するヒープのピークを報告する
// ため、64 MiB の目標との比較と、ケース間の比較による「追加ヒープが投稿件数では
// なくページサイズに従う」ことの確認ができる。
func BenchmarkBuilder_StreamsArchive(b *testing.B) {
	validateBenchPostContent(b)

	pageSize := int64(usecase.ExportPostPageSize)
	cases := []benchArchiveCase{
		{postCount: 100_000, pageSize: pageSize},
		{postCount: 100_000, pageSize: pageSize / 10},
		{postCount: 100_000, pageSize: pageSize * 10},
		{postCount: 10_000, pageSize: pageSize},
	}

	for _, benchCase := range cases {
		b.Run(fmt.Sprintf("posts=%d/page=%d", benchCase.postCount, benchCase.pageSize), func(b *testing.B) {
			archive := benchArchive(b, benchCase.postCount)

			var peakHeap, archiveBytes int64
			for b.Loop() {
				iterationHeap, iterationBytes := writeBenchArchive(b, archive, benchCase.pageSize)
				peakHeap = max(peakHeap, iterationHeap)
				archiveBytes = iterationBytes
			}

			b.ReportMetric(float64(peakHeap)/(1<<20), "peak_heap_MiB")
			b.ReportMetric(float64(archiveBytes)/(1<<20), "zip_MiB")
			if peakHeap > benchMaxHeapBytes {
				b.Errorf("追加ヒープのピーク = %d バイト, want <= %d バイト", peakHeap, benchMaxHeapBytes)
			}
		})
	}
}

// benchArchive declares an archive holding postCount posts spread over the
// benchmark's months.
//
// [Ja] benchArchive は postCount 件の投稿をベンチマークの月数に分散して持つ
// アーカイブを宣言する。
func benchArchive(b *testing.B, postCount int64) usecase.ExportArchive {
	b.Helper()

	months := make([]usecase.ExportArchiveMonth, 0, benchArchiveMonths)
	for i := range benchArchiveMonths {
		count := postCount / benchArchiveMonths
		if i < int(postCount%benchArchiveMonths) {
			count++
		}
		months = append(months, newMonth(2023+i/12, time.Month(i%12+1), count))
	}
	return newArchive(b, months...)
}

// writeBenchArchive builds one archive the way the generation flow will: posts
// are read one page at a time and handed to the writer, and nothing but the
// current page is kept. The peak heap is sampled once per page, which is where
// a page is fully materialized, and the archive is written to a counter rather
// than a buffer so that only the builder's own memory is measured.
//
// [Ja] writeBenchArchive は生成処理と同じ形でアーカイブを 1 つ構築する。投稿は
// 1 ページずつ取得して writer へ渡し、現在のページ以外は保持しない。ヒープの
// ピークはページが完全に materialize される 1 ページごとに標本化し、アーカイブは
// バッファではなくカウンターへ書き出して builder 自身のメモリだけを測る。
func writeBenchArchive(b *testing.B, archive usecase.ExportArchive, pageSize int64) (peakHeap, archiveBytes int64) {
	b.Helper()

	ctx := context.Background()
	counter := &countingWriter{}

	runtime.GC()
	var base runtime.MemStats
	runtime.ReadMemStats(&base)

	writer := exportfile.NewBuilder().NewArchive(counter, archive)
	if err := writer.WriteIndex(ctx); err != nil {
		b.Fatalf("index.html の書き出しに失敗: %v", err)
	}
	for _, month := range archive.Months {
		monthWriter, err := writer.OpenMonth(ctx, month)
		if err != nil {
			b.Fatalf("月のエントリの作成に失敗: %v", err)
		}
		for written := int64(0); written < month.PostCount; {
			page := benchPostPage(month, written, min(pageSize, month.PostCount-written))
			for _, post := range page {
				if err := monthWriter.WritePost(ctx, post); err != nil {
					b.Fatalf("投稿の書き出しに失敗: %v", err)
				}
			}
			written += int64(len(page))
			pageHeap := addedHeap(base)
			runtime.KeepAlive(page)
			peakHeap = max(peakHeap, pageHeap)
		}
		if err := monthWriter.Close(); err != nil {
			b.Fatalf("月のエントリのクローズに失敗: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		b.Fatalf("アーカイブのクローズに失敗: %v", err)
	}
	return peakHeap, counter.written
}

// validateBenchPostContent checks the fixture's stated worst-case size before
// its measurements are used to select generation limits.
//
// [Ja] validateBenchPostContent は生成上限の選定に計測値を使う前に、フィクスチャが
// 想定する最大サイズであることを確認する。
func validateBenchPostContent(b *testing.B) {
	b.Helper()

	month := newMonth(2023, time.January, 1)
	content := benchPostPage(month, 0, 1)[0].Content
	if got := utf8.RuneCountInString(content); got != benchPostContentRunes {
		b.Fatalf("ベンチマーク投稿の文字数 = %d, want %d", got, benchPostContentRunes)
	}
	if got := len(content); got != benchPostContentBytes {
		b.Fatalf("ベンチマーク投稿のバイト数 = %d, want %d", got, benchPostContentBytes)
	}
}

// benchPostPage builds one page of a month's posts, standing in for the page
// the repository returns.
//
// [Ja] benchPostPage は月の投稿を 1 ページ分組み立て、repository が返すページの
// 代わりになる。
func benchPostPage(month usecase.ExportArchiveMonth, offset, count int64) []usecase.ExportArchivePost {
	posts := make([]usecase.ExportArchivePost, 0, count)
	for i := offset; i < offset+count; i++ {
		start := int(i*benchPostContentRunes) % (len(benchFiller) - benchPostContentRunes)
		posts = append(posts, usecase.ExportArchivePost{
			ID:          fmt.Sprintf("01JXXXXXXXXXXXXXXXXXXX%04d", i),
			Content:     string(benchFiller[start : start+benchPostContentRunes]),
			PublishedAt: month.LocalMonthStart.Add(time.Duration(i) * time.Minute),
		})
	}
	return posts
}

// addedHeap returns how much the allocated heap has grown since base. The
// figure is HeapAlloc, which counts unswept garbage alongside reachable
// objects, so it bounds what the build retains rather than reporting it
// exactly. A shrinking heap reads as zero rather than wrapping around the
// unsigned counters.
//
// [Ja] addedHeap は base から割り当て済みヒープがどれだけ増えたかを返す。値は
// HeapAlloc で、到達可能なオブジェクトに加えて未回収のガベージも含むため、生成が
// 保持する量そのものではなくその上限として読む。ヒープが減った場合は符号なし
// カウンターを回り込ませず 0 として扱う。
func addedHeap(base runtime.MemStats) int64 {
	var current runtime.MemStats
	runtime.ReadMemStats(&current)

	if current.HeapAlloc <= base.HeapAlloc {
		return 0
	}
	return int64(current.HeapAlloc - base.HeapAlloc)
}
