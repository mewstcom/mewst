// Package exportfileはユーザーがエクスポートからダウンロードするzip
// アーカイブ (index.htmlと月ごとのHTMLファイル) を構築する。
// usecase.ExportArchiveBuilder portの実装で、各エントリを渡されたwriterへ
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
	// indexEntryNameはアーカイブの目次、monthEntryFormatは1か月分の
	// エントリ。月の部分は暦月そのもののため、ユーザー入力をサニタイズせずに
	// エントリ名が一意になる。
	indexEntryName   = "index.html"
	monthEntryFormat = "posts/%s.html"

	// monthLayoutはエントリ名と見出しに使う暦月の書式。
	monthLayout = "2006-01"
)

// Builderはアーカイブのwriterを生成する。自身は状態を持たないため、
// 1インスタンスを全ての生成ジョブで共有する。
type Builder struct{}

// Builderは生成UseCaseが依存するportを満たす必要がある。
var _ usecase.ExportArchiveBuilder = (*Builder)(nil)

// NewBuilderはBuilderを生成する。
func NewBuilder() *Builder {
	return &Builder{}
}

// NewArchiveはwへ書き出すアーカイブを開始する。
func (b *Builder) NewArchive(w io.Writer, archive usecase.ExportArchive) usecase.ExportArchiveWriter {
	pendingMonths := make(map[string]usecase.ExportArchiveMonth, len(archive.Months))
	var declarationErr error
	for _, month := range archive.Months {
		name := monthEntryName(month)
		if _, exists := pendingMonths[name]; exists {
			// 月の宣言が重複するとindex.htmlには同じリンクが2回出る一方、
			// 開けるzipエントリは1つだけになる。map内で黙って畳み込まず、
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

// archiveWriterは1つのアーカイブを書き出す。pendingMonthsは
// index.htmlがリンクする月から始まり、書き出すたびにその月を失う。これにより
// 目次に無い月のエントリも、同じ月の2つ目のエントリも拒否され、目次と
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

// WriteIndexは宣言済みの月からindex.htmlを書き出す。
func (a *archiveWriter) WriteIndex(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if a.declarationErr != nil {
		return a.declarationErr
	}
	if a.indexWritten {
		return errors.New("index.htmlは既に書き出し済み")
	}

	w, err := a.createEntry(indexEntryName)
	if err != nil {
		return err
	}
	// ここから先はエントリが存在するため、indexWrittenは2回目の
	// WriteIndexが重複したエントリを追加することも止める。本文の失敗は代わりに
	// entryErrとして保持し、下位writerがエラーを返し続けることに頼らず、
	// Close自身が切り詰められたindexを報告できるようにする。
	a.indexWritten = true

	if err := writeIndexHTML(ctx, w, a.archive); err != nil {
		a.entryErr = errors.Join(a.entryErr, fmt.Errorf("%sの書き出しに失敗: %w", indexEntryName, err))
		return a.entryErr
	}
	return nil
}

// OpenMonthはmonthのエントリを開始し、その投稿用のwriterを返す。
func (a *archiveWriter) OpenMonth(ctx context.Context, month usecase.ExportArchiveMonth) (usecase.ExportArchiveMonthWriter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if a.declarationErr != nil {
		return nil, a.declarationErr
	}
	if !a.indexWritten {
		return nil, errors.New("index.htmlがまだ書き出されていない")
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

	// ヘッダーを書き出すまで月は未処理のままにする。ここで失敗した場合、
	// 不完全なエントリが残ることをCloseが未出力の月として報告できる。
	if err := writeMonthHeaderHTML(ctx, w, a.archive, declaredMonth); err != nil {
		return nil, fmt.Errorf("%sの書き出しに失敗: %w", name, err)
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

// Closeは開いたままの月のエントリを閉じてからアーカイブを完成させる。
// 不正なアーカイブを成功扱いせず、かつ出力ストリームを確実に解放できるよう、
// zip writer自体は常に閉じ、その後でクローズと完全性の全エラーをまとめて返す。
// 2回目以降の呼び出しは何もしないため、呼び出し側はdeferしつつ、エラーを
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
		closeErrs = append(closeErrs, errors.New("index.htmlが書き出されていない"))
	}
	if len(a.pendingMonths) > 0 {
		closeErrs = append(closeErrs, fmt.Errorf("目次に対応する月のエントリが書き出されていない (%d件)", len(a.pendingMonths)))
	}
	if err := a.zip.Close(); err != nil {
		closeErrs = append(closeErrs, fmt.Errorf("アーカイブのクローズに失敗: %w", err))
	}
	return errors.Join(closeErrs...)
}

// createEntryは生成時刻を記録したdeflate圧縮のエントリを追加する。
func (a *archiveWriter) createEntry(name string) (io.Writer, error) {
	w, err := a.zip.CreateHeader(&zip.FileHeader{
		Name:     name,
		Method:   zip.Deflate,
		Modified: a.archive.GeneratedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("zipエントリの作成に失敗 (entry: %s): %w", name, err)
	}
	return w, nil
}

// monthWriterは1か月分のエントリへ投稿を追記する。エントリのwriterは
// zip writerに属し、次のエントリを作るまでしか有効でないため、閉じた月への
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

// WritePostは月のエントリへ投稿を1件追記する。
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
		m.entryErr = fmt.Errorf("%sの書き出しに失敗: %w", m.name, err)
		return m.entryErr
	}
	m.writtenPostCount++
	return nil
}

// Closeは月のエントリを完成させ、書き出した投稿数がindex.htmlの
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
		closeErrs = append(closeErrs, fmt.Errorf("%sの書き出しに失敗: %w", m.name, err))
	}

	closeErr := errors.Join(closeErrs...)
	if closeErr != nil {
		m.parent.entryErr = errors.Join(m.parent.entryErr, closeErr)
	}
	return closeErr
}

// monthEntryNameは月のエントリ名 (例: posts/2026-07.html) を返す。
func monthEntryName(month usecase.ExportArchiveMonth) string {
	return fmt.Sprintf(monthEntryFormat, monthLabel(month))
}

// monthLabelは暦月をYYYY-MMで返す。LocalMonthStartは時点ではなく
// ラベルのため、ゾーンを変換せずに書式化する。
func monthLabel(month usecase.ExportArchiveMonth) string {
	return month.LocalMonthStart.Format(monthLayout)
}
