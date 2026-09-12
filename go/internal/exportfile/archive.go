// Package exportfile builds the zip archive a user downloads from an export:
// index.html plus one HTML file per month. It implements the
// usecase.ExportArchiveBuilder port and writes every entry straight to the
// writer it is given, so neither a month nor the archive is held in memory.
//
// [Ja] exportfile パッケージはユーザーがエクスポートからダウンロードする zip
// アーカイブ (index.html と月ごとの HTML ファイル) を構築する。
// usecase.ExportArchiveBuilder port の実装で、各エントリを渡された writer へ
// そのまま書き出すため、月全体もアーカイブ全体もメモリに保持しない。
package exportfile

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/mewstcom/mewst/go/internal/usecase"
)

const (
	// indexEntryName is the archive's table of contents, and monthEntryFormat
	// is the entry of one month. The month part is the calendar month itself,
	// so entry names are unique without sanitizing user input.
	//
	// [Ja] indexEntryName はアーカイブの目次、monthEntryFormat は 1 か月分の
	// エントリ。月の部分は暦月そのもののため、ユーザー入力をサニタイズせずに
	// エントリ名が一意になる。
	indexEntryName   = "index.html"
	monthEntryFormat = "posts/%s.html"

	// monthLayout formats a calendar month for entry names and headings.
	//
	// [Ja] monthLayout はエントリ名と見出しに使う暦月の書式。
	monthLayout = "2006-01"
)

// Builder creates archive writers. It holds no state of its own, so one
// instance is shared by every generation job.
//
// [Ja] Builder はアーカイブの writer を生成する。自身は状態を持たないため、
// 1 インスタンスを全ての生成ジョブで共有する。
type Builder struct{}

// Builder must satisfy the port the generation use case depends on.
//
// [Ja] Builder は生成 UseCase が依存する port を満たす必要がある。
var _ usecase.ExportArchiveBuilder = (*Builder)(nil)

// NewBuilder creates a Builder.
//
// [Ja] NewBuilder は Builder を生成する。
func NewBuilder() *Builder {
	return &Builder{}
}

// NewArchive starts an archive written to w.
//
// [Ja] NewArchive は w へ書き出すアーカイブを開始する。
func (b *Builder) NewArchive(w io.Writer, archive usecase.ExportArchive) usecase.ExportArchiveWriter {
	pendingMonths := make(map[string]usecase.ExportArchiveMonth, len(archive.Months))
	var declarationErr error
	for _, month := range archive.Months {
		name := monthEntryName(month)
		if _, exists := pendingMonths[name]; exists {
			// A duplicate declaration would write the same link twice in index.html
			// while only one zip entry could be opened, so remember it as an invalid
			// archive instead of silently collapsing it in the map.
			//
			// [Ja] 月の宣言が重複すると index.html には同じリンクが 2 回出る一方、
			// 開ける zip エントリは 1 つだけになる。map 内で黙って畳み込まず、
			// 不正なアーカイブとして記録する。
			if declarationErr == nil {
				declarationErr = fmt.Errorf("同じ月が複数回宣言されている (entry: %s)", name)
			}
			continue
		}
		pendingMonths[name] = month
	}

	return &archiveWriter{
		zip:            zip.NewWriter(w),
		archive:        archive,
		pendingMonths:  pendingMonths,
		declarationErr: declarationErr,
	}
}

// archiveWriter writes one archive. pendingMonths starts as the months
// index.html links to and loses each month as it is written, so an entry that
// the index does not list, and a second entry for the same month, are both
// rejected instead of producing an archive whose index does not match its
// files.
//
// [Ja] archiveWriter は 1 つのアーカイブを書き出す。pendingMonths は
// index.html がリンクする月から始まり、書き出すたびにその月を失う。これにより
// 目次に無い月のエントリも、同じ月の 2 つ目のエントリも拒否され、目次と
// ファイルが食い違うアーカイブができない。
type archiveWriter struct {
	zip            *zip.Writer
	archive        usecase.ExportArchive
	pendingMonths  map[string]usecase.ExportArchiveMonth
	indexWritten   bool
	openMonth      *monthWriter
	declarationErr error
	entryErr       error
	closed         bool
}

// WriteIndex writes index.html from the declared months.
//
// [Ja] WriteIndex は宣言済みの月から index.html を書き出す。
func (a *archiveWriter) WriteIndex(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if a.declarationErr != nil {
		return a.declarationErr
	}
	if a.indexWritten {
		return errors.New("index.html は既に書き出し済み")
	}

	w, err := a.createEntry(indexEntryName)
	if err != nil {
		return err
	}
	// The entry exists from here on, so indexWritten also stops a second
	// WriteIndex from adding a duplicate one. A failed body is kept as entryErr
	// instead, so that Close reports the truncated index on its own rather than
	// relying on the underlying writer to keep returning its error.
	//
	// [Ja] ここから先はエントリが存在するため、indexWritten は 2 回目の
	// WriteIndex が重複したエントリを追加することも止める。本文の失敗は代わりに
	// entryErr として保持し、下位 writer がエラーを返し続けることに頼らず、
	// Close 自身が切り詰められた index を報告できるようにする。
	a.indexWritten = true

	if err := writeIndexHTML(ctx, w, a.archive); err != nil {
		a.entryErr = errors.Join(a.entryErr, fmt.Errorf("%s の書き出しに失敗: %w", indexEntryName, err))
		return a.entryErr
	}
	return nil
}

// OpenMonth starts the entry for month and returns the writer for its posts.
//
// [Ja] OpenMonth は month のエントリを開始し、その投稿用の writer を返す。
func (a *archiveWriter) OpenMonth(ctx context.Context, month usecase.ExportArchiveMonth) (usecase.ExportArchiveMonthWriter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if a.declarationErr != nil {
		return nil, a.declarationErr
	}
	if !a.indexWritten {
		return nil, errors.New("index.html がまだ書き出されていない")
	}
	if a.openMonth != nil {
		return nil, errors.New("前の月のエントリが閉じられていない")
	}

	name := monthEntryName(month)
	declaredMonth, ok := a.pendingMonths[name]
	if !ok {
		return nil, fmt.Errorf("目次に含まれない、または書き出し済みの月 (entry: %s)", name)
	}

	w, err := a.createEntry(name)
	if err != nil {
		return nil, err
	}

	// The month stays pending until its header is written, so a failure here
	// leaves an incomplete entry that Close still reports as a missing month.
	//
	// [Ja] ヘッダーを書き出すまで月は未処理のままにする。ここで失敗した場合、
	// 不完全なエントリが残ることを Close が未出力の月として報告できる。
	if err := writeMonthHeaderHTML(ctx, w, a.archive, declaredMonth); err != nil {
		return nil, fmt.Errorf("%s の書き出しに失敗: %w", name, err)
	}
	delete(a.pendingMonths, name)

	a.openMonth = &monthWriter{
		parent:            a,
		name:              name,
		w:                 w,
		expectedPostCount: declaredMonth.PostCount,
	}
	return a.openMonth, nil
}

// Close finishes the archive, closing an open month entry first. It always
// closes the underlying zip writer, then returns every finalization and
// completeness error together so an invalid archive is never reported as a
// successful result and the output stream is still released. Calling it twice
// is a no-op, so a caller can defer it and still close explicitly on the path
// where the error matters.
//
// [Ja] Close は開いたままの月のエントリを閉じてからアーカイブを完成させる。
// 不正なアーカイブを成功扱いせず、かつ出力ストリームを確実に解放できるよう、
// zip writer 自体は常に閉じ、その後でクローズと完全性の全エラーをまとめて返す。
// 2 回目以降の呼び出しは何もしないため、呼び出し側は defer しつつ、エラーを
// 見たい経路では明示的に閉じることもできる。
func (a *archiveWriter) Close() error {
	if a.closed {
		return nil
	}
	a.closed = true

	var closeErrs []error

	if a.openMonth != nil {
		_ = a.openMonth.Close()
	}
	if a.declarationErr != nil {
		closeErrs = append(closeErrs, a.declarationErr)
	}
	if a.entryErr != nil {
		closeErrs = append(closeErrs, a.entryErr)
	}
	if !a.indexWritten {
		closeErrs = append(closeErrs, errors.New("index.html が書き出されていない"))
	}
	if len(a.pendingMonths) > 0 {
		closeErrs = append(closeErrs, fmt.Errorf("目次に対応する月のエントリが書き出されていない (%d 件)", len(a.pendingMonths)))
	}
	if err := a.zip.Close(); err != nil {
		closeErrs = append(closeErrs, fmt.Errorf("アーカイブのクローズに失敗: %w", err))
	}
	return errors.Join(closeErrs...)
}

// createEntry adds a deflated entry stamped with the archive's generation time.
//
// [Ja] createEntry は生成時刻を記録した deflate 圧縮のエントリを追加する。
func (a *archiveWriter) createEntry(name string) (io.Writer, error) {
	w, err := a.zip.CreateHeader(&zip.FileHeader{
		Name:     name,
		Method:   zip.Deflate,
		Modified: a.archive.GeneratedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("zip エントリの作成に失敗 (entry: %s): %w", name, err)
	}
	return w, nil
}

// monthWriter appends posts to one month's entry. The entry writer belongs to
// the zip writer and stays valid only until the next entry is created, so
// writing to a closed month is rejected rather than corrupting the next entry.
//
// [Ja] monthWriter は 1 か月分のエントリへ投稿を追記する。エントリの writer は
// zip writer に属し、次のエントリを作るまでしか有効でないため、閉じた月への
// 書き込みは次のエントリを壊さないよう拒否する。
type monthWriter struct {
	parent            *archiveWriter
	name              string
	w                 io.Writer
	expectedPostCount int64
	writtenPostCount  int64
	entryErr          error
	closed            bool
}

// WritePost appends one post to the month's entry.
//
// [Ja] WritePost は月のエントリへ投稿を 1 件追記する。
func (m *monthWriter) WritePost(ctx context.Context, post usecase.ExportArchivePost) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if m.closed {
		return fmt.Errorf("閉じた月のエントリには追記できない (entry: %s)", m.name)
	}
	if m.entryErr != nil {
		return m.entryErr
	}
	if m.writtenPostCount >= m.expectedPostCount {
		m.entryErr = fmt.Errorf(
			"目次の宣言件数を超える投稿は書き出せない (entry: %s, declared: %d)",
			m.name,
			m.expectedPostCount,
		)
		return m.entryErr
	}

	if err := writePostHTML(ctx, m.w, m.parent.archive, post); err != nil {
		m.entryErr = fmt.Errorf("%s の書き出しに失敗: %w", m.name, err)
		return m.entryErr
	}
	m.writtenPostCount++
	return nil
}

// Close finishes the month's entry and verifies that the number of posts
// written matches the count declared in index.html.
//
// [Ja] Close は月のエントリを完成させ、書き出した投稿数が index.html の
// 宣言件数と一致することを検証する。
func (m *monthWriter) Close() error {
	if m.closed {
		return nil
	}
	m.closed = true
	m.parent.openMonth = nil

	var closeErrs []error
	if m.entryErr != nil {
		closeErrs = append(closeErrs, m.entryErr)
	}
	if m.writtenPostCount != m.expectedPostCount {
		closeErrs = append(closeErrs, fmt.Errorf(
			"目次の宣言件数と書き出した投稿数が一致しない (entry: %s, declared: %d, written: %d)",
			m.name,
			m.expectedPostCount,
			m.writtenPostCount,
		))
	}
	if err := writeMonthFooterHTML(m.w); err != nil {
		closeErrs = append(closeErrs, fmt.Errorf("%s の書き出しに失敗: %w", m.name, err))
	}

	closeErr := errors.Join(closeErrs...)
	if closeErr != nil {
		m.parent.entryErr = errors.Join(m.parent.entryErr, closeErr)
	}
	return closeErr
}

// monthEntryName returns the entry name of a month, for example
// posts/2026-07.html.
//
// [Ja] monthEntryName は月のエントリ名 (例: posts/2026-07.html) を返す。
func monthEntryName(month usecase.ExportArchiveMonth) string {
	return fmt.Sprintf(monthEntryFormat, monthLabel(month))
}

// monthLabel returns the calendar month as YYYY-MM. LocalMonthStart is a label
// rather than an instant, so it is formatted without converting its zone.
//
// [Ja] monthLabel は暦月を YYYY-MM で返す。LocalMonthStart は時点ではなく
// ラベルのため、ゾーンを変換せずに書式化する。
func monthLabel(month usecase.ExportArchiveMonth) string {
	return month.LocalMonthStart.Format(monthLayout)
}
